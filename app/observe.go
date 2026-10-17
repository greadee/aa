package main

import (
	"context"
	"io"

	"github.com/greadee/aa/app/internal/controlplane"
	"github.com/greadee/aa/app/internal/obsplane"
	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/obsv/report"
)

// observationBackend adapts the composed obsv plane to the control-plane
// transport types.
type observationBackend struct {
	plane *obsplane.Plane
}

func (o observationBackend) Replay(sessionID string, from int) ([]controlplane.Observation, error) {
	if o.plane == nil {
		return nil, nil
	}
	events, err := o.plane.Replay(sessionID, from)
	if err != nil {
		return nil, err
	}
	out := make([]controlplane.Observation, 0, len(events))
	for _, ev := range events {
		out = append(out, observationFromEvent(ev))
	}
	return out, nil
}

func (o observationBackend) Report(sessionID string) (controlplane.ObservationReport, error) {
	if o.plane == nil {
		return controlplane.ObservationReport{}, nil
	}
	rep, err := o.plane.Report(sessionID)
	if err != nil {
		return controlplane.ObservationReport{}, err
	}
	return observationReport(rep), nil
}

func (o observationBackend) Subscribe(ctx context.Context, sessionID string, from int) (<-chan controlplane.Observation, error) {
	if o.plane == nil {
		out := make(chan controlplane.Observation)
		close(out)
		return out, nil
	}
	sub, err := o.plane.Subscribe(sessionID, from)
	if err != nil {
		return nil, err
	}
	out := make(chan controlplane.Observation, 64)
	go func() {
		defer close(out)
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Events():
				if !ok {
					return
				}
				select {
				case out <- observationFromEvent(ev):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (o observationBackend) IngestCodex(sessionID string, r io.Reader) (controlplane.CodexIngest, error) {
	if o.plane == nil {
		return controlplane.CodexIngest{}, nil
	}
	ingested, err := o.plane.IngestCodex(sessionID, r)
	if err != nil {
		return controlplane.CodexIngest{}, err
	}
	return controlplane.CodexIngest{Accepted: ingested.Accepted, Dropped: ingested.Dropped}, nil
}

func observationFromEvent(ev protocol.Event) controlplane.Observation {
	return controlplane.Observation{
		Version:       ev.Version,
		SessionID:     ev.SessionID,
		Sequence:      ev.Sequence,
		OccurredAt:    ev.OccurredAt,
		SourceType:    string(ev.SourceType),
		Source:        ev.Source,
		Action:        ev.Action,
		Confidence:    string(ev.Confidence),
		Actor:         ev.Actor,
		WorkPackageID: ev.WorkPackageID,
		AttemptID:     ev.AttemptID,
		Metadata:      ev.Metadata,
	}
}

func observationReport(rep report.Report) controlplane.ObservationReport {
	out := controlplane.ObservationReport{
		SessionID:     rep.SessionID,
		Events:        rep.Events,
		FirstSequence: rep.FirstSequence,
		LastSequence:  rep.LastSequence,
		WorkPackages:  rep.WorkPackages,
		Actors:        rep.Actors,
	}
	out.BySourceType = make([]controlplane.ObservationCount, 0, len(rep.BySourceType))
	for _, c := range rep.BySourceType {
		out.BySourceType = append(out.BySourceType, controlplane.ObservationCount{Label: c.Label, Count: c.Count})
	}
	out.ByConfidence = make([]controlplane.ObservationCount, 0, len(rep.ByConfidence))
	for _, c := range rep.ByConfidence {
		out.ByConfidence = append(out.ByConfidence, controlplane.ObservationCount{Label: c.Label, Count: c.Count})
	}
	return out
}

// emit journals one observation. It is intentionally failure-open: the result
// is discarded so observation can never alter control-plane behavior.
func (b *backend) emit(sessionID string, ev protocol.Event) {
	if b.g.obs == nil {
		return
	}
	_ = b.g.obs.Emit(sessionID, ev)
}
