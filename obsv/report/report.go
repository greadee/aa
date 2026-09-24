// Package report derives deterministic work reports from journaled observation.
//
// A report depends only on the events it is given. Events are ordered by
// sequence, never by arrival order or a wall clock, so the same evidence always
// produces the same report.
package report

import (
	"sort"

	"github.com/greadee/aa/obsv/protocol"
)

// Count is a labelled event count.
type Count struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Report summarizes one observation session.
type Report struct {
	SessionID     string   `json:"sessionId"`
	Events        int      `json:"events"`
	FirstSequence int      `json:"firstSequence"`
	LastSequence  int      `json:"lastSequence"`
	BySourceType  []Count  `json:"bySourceType"`
	ByConfidence  []Count  `json:"byConfidence"`
	WorkPackages  []string `json:"workPackages,omitempty"`
	Actors        []string `json:"actors,omitempty"`
}

// Build returns a deterministic report for a session.
func Build(sessionID string, events []protocol.Event) Report {
	ordered := make([]protocol.Event, len(events))
	copy(ordered, events)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })

	r := Report{SessionID: sessionID, Events: len(ordered)}
	if len(ordered) > 0 {
		r.FirstSequence = ordered[0].Sequence
		r.LastSequence = ordered[len(ordered)-1].Sequence
	}

	types := make(map[string]int)
	confidences := make(map[string]int)
	packages := make(map[string]bool)
	actors := make(map[string]bool)
	for _, ev := range ordered {
		types[string(ev.SourceType)]++
		confidences[string(ev.Confidence)]++
		if ev.WorkPackageID != "" {
			packages[ev.WorkPackageID] = true
		}
		if ev.Actor != "" {
			actors[ev.Actor] = true
		}
	}
	r.BySourceType = toCounts(types)
	r.ByConfidence = toCounts(confidences)
	r.WorkPackages = toSorted(packages)
	r.Actors = toSorted(actors)
	return r
}

func toCounts(m map[string]int) []Count {
	out := make([]Count, 0, len(m))
	for label, count := range m {
		out = append(out, Count{Label: label, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

func toSorted(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
