package scheduler

import (
	"context"

	"github.com/greadee/aa/kernel/allocator/planner"
)

// RunReport aggregates one scheduler run.
type RunReport struct {
	ProjectID  string         `json:"projectId"`
	Dispatched int            `json:"dispatched"`
	Accepted   int            `json:"accepted"`
	Failed     int            `json:"failed"`
	Pending    int            `json:"pending"`
	Attempts   map[string]int `json:"attempts"`
}

// Run dispatches ready work packages until none remain, bounded by
// Config.MaxConcurrency and retrying failures up to Config.MaxAttempts.
//
// It is deterministic: the ready set is dispatched in sorted order and the
// aggregate does not depend on completion timing or goroutine scheduling.
// Cancelling ctx stops dispatching and propagates to in-flight runtime calls.
// A failing work package does not abort independent work (partial failure).
func (o *Scheduler) Run(ctx context.Context) (RunReport, error) {
	report := RunReport{ProjectID: o.projectID, Attempts: map[string]int{}}
	concurrency := o.cfg.MaxConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	maxAttempts := o.cfg.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		o.mu.Lock()
		ready := o.graph.Ready()
		o.mu.Unlock()
		if len(ready) == 0 {
			break
		}
		batch := ready
		if len(batch) > concurrency {
			batch = batch[:concurrency]
		}

		type outcome struct {
			wp  string
			rep DispatchReport
			err error
		}
		outcomes := make(chan outcome, len(batch))
		for _, id := range batch {
			go func(id string) {
				rep, err := o.dispatchOne(ctx, id)
				outcomes <- outcome{wp: id, rep: rep, err: err}
			}(id)
		}

		for range batch {
			out := <-outcomes
			if out.err != nil {
				return report, out.err
			}
			report.Dispatched++
			report.Attempts[out.wp]++
			switch out.rep.Status {
			case "succeeded":
				report.Accepted++
			case "failed":
				if report.Attempts[out.wp] < maxAttempts && o.resetForRetry(out.wp) {
					continue
				}
				report.Failed++
			default:
				report.Pending++
			}
		}
	}
	return report, nil
}

// resetForRetry returns a failed work package to the ready state for another
// attempt. It reports false when the package cannot be retried.
func (o *Scheduler) resetForRetry(workPackageID string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.assignments[workPackageID]; !ok {
		return false
	}
	return o.graph.SetState(workPackageID, planner.StateReady) == nil
}
