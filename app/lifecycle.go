package main

import (
	"time"

	"github.com/greadee/aa/app/internal/controlplane"
	"github.com/greadee/aa/app/internal/obsplane"
	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/runtime/lifecycle"
)

// lifecycleBackend adapts the composed lifecycle manager to the control-plane
// transport types.
type lifecycleBackend struct {
	manager *lifecycle.Manager
}

func (l lifecycleBackend) Workers() []controlplane.WorkerLifecycle {
	if l.manager == nil {
		return nil
	}
	snapshots := l.manager.Workers()
	out := make([]controlplane.WorkerLifecycle, 0, len(snapshots))
	for _, s := range snapshots {
		worker := controlplane.WorkerLifecycle{
			ID:             s.ID,
			SelectedWorker: s.SelectedWorker,
			AssignmentID:   s.AssignmentID,
			WorkPackageID:  s.WorkPackageID,
			Capabilities:   s.Capabilities,
			State:          string(s.State),
			Status:         s.Status,
			Summary:        s.Summary,
			Error:          s.Error,
		}
		if !s.CreatedAt.IsZero() {
			worker.CreatedAt = s.CreatedAt.UTC().Format(time.RFC3339Nano)
		}
		if !s.EndedAt.IsZero() {
			worker.EndedAt = s.EndedAt.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, worker)
	}
	return out
}

func (l lifecycleBackend) WorkerEvents() []controlplane.WorkerLifecycleEvent {
	if l.manager == nil {
		return nil
	}
	events := l.manager.Events()
	out := make([]controlplane.WorkerLifecycleEvent, 0, len(events))
	for _, ev := range events {
		event := controlplane.WorkerLifecycleEvent{
			WorkerID:      ev.WorkerID,
			AssignmentID:  ev.AssignmentID,
			WorkPackageID: ev.WorkPackageID,
			State:         string(ev.State),
			Reason:        ev.Reason,
		}
		if !ev.At.IsZero() {
			event.At = ev.At.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, event)
	}
	return out
}

// lifecycleObserver forwards lifecycle transitions to the observation
// substrate. Emission is failure-open, so a broken observer never changes a
// worker run.
type lifecycleObserver struct {
	plane *obsplane.Plane
}

func (o lifecycleObserver) Emit(ev lifecycle.Event) {
	if o.plane == nil {
		return
	}
	metadata := map[string]string{
		"kind":   "worker",
		"status": string(ev.State),
		"target": ev.WorkerID,
	}
	if ev.Reason != "" {
		metadata["outcome"] = ev.Reason
	}
	_ = o.plane.Emit("runtime", protocol.Event{
		SourceType:    protocol.SourceSession,
		Confidence:    protocol.ConfidenceObserved,
		Action:        "worker.lifecycle",
		Actor:         "runtime",
		WorkPackageID: ev.WorkPackageID,
		Metadata:      metadata,
	})
}
