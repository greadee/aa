package query

import (
	"encoding/json"
	"sort"
)

// event is the minimal event shape needed to derive histories.
type event struct {
	ID         string `json:"id"`
	Sequence   int    `json:"sequence"`
	Type       string `json:"type"`
	OccurredAt string `json:"occurredAt"`
	ProjectID  string `json:"projectId"`
	Aggregate  struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"aggregate"`
	Actor struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"actor"`
	Payload map[string]any `json:"payload"`
}

// WorkHistoryEntry summarizes a work package's lifecycle across events.
type WorkHistoryEntry struct {
	WorkPackageID string
	CreatedAt     string
	ReadyAt       string
	CompletedAt   string
	State         string
	EventIDs      []string
}

// ProjectHistoryEntry is a project-level event in sequence order.
type ProjectHistoryEntry struct {
	ProjectID  string
	EventID    string
	Type       string
	Sequence   int
	OccurredAt string
}

// JobHistoryEntry summarizes an assignment's execution span.
type JobHistoryEntry struct {
	AssignmentID  string
	WorkPackageID string
	StartedAt     string
	FinishedAt    string
	Outcome       string
	EventIDs      []string
}

func (q *Query) decodedEvents() ([]event, error) {
	records, err := q.store.Events()
	if err != nil {
		return nil, err
	}
	events := make([]event, 0, len(records))
	for _, rec := range records {
		var e event
		if err := json.Unmarshal(rec.Data, &e); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })
	return events, nil
}

// WorkHistory derives a per-work-package lifecycle from events.
func (q *Query) WorkHistory() ([]WorkHistoryEntry, error) {
	events, err := q.decodedEvents()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*WorkHistoryEntry)
	for _, e := range events {
		if e.Aggregate.Kind != "work_package" {
			continue
		}
		entry, ok := byID[e.Aggregate.ID]
		if !ok {
			entry = &WorkHistoryEntry{WorkPackageID: e.Aggregate.ID, State: "proposed"}
			byID[e.Aggregate.ID] = entry
		}
		entry.EventIDs = append(entry.EventIDs, e.ID)
		switch e.Type {
		case "WORK_PACKAGE_CREATED":
			entry.CreatedAt = e.OccurredAt
		case "WORK_PACKAGE_READY":
			entry.ReadyAt = e.OccurredAt
			if entry.State == "proposed" {
				entry.State = "ready"
			}
		case "WORK_PACKAGE_COMPLETED":
			entry.CompletedAt = e.OccurredAt
			entry.State = "completed"
		}
	}
	out := make([]WorkHistoryEntry, 0, len(byID))
	for _, entry := range byID {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WorkPackageID < out[j].WorkPackageID })
	return out, nil
}

// ProjectHistory returns project aggregate events in sequence order.
func (q *Query) ProjectHistory() ([]ProjectHistoryEntry, error) {
	events, err := q.decodedEvents()
	if err != nil {
		return nil, err
	}
	var out []ProjectHistoryEntry
	for _, e := range events {
		if e.Aggregate.Kind != "project" {
			continue
		}
		out = append(out, ProjectHistoryEntry{
			ProjectID:  e.ProjectID,
			EventID:    e.ID,
			Type:       e.Type,
			Sequence:   e.Sequence,
			OccurredAt: e.OccurredAt,
		})
	}
	return out, nil
}

// JobHistory derives per-assignment execution spans from events.
func (q *Query) JobHistory() ([]JobHistoryEntry, error) {
	events, err := q.decodedEvents()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*JobHistoryEntry)
	for _, e := range events {
		if e.Aggregate.Kind != "assignment" {
			continue
		}
		entry, ok := byID[e.Aggregate.ID]
		if !ok {
			entry = &JobHistoryEntry{AssignmentID: e.Aggregate.ID, Outcome: "unknown"}
			byID[e.Aggregate.ID] = entry
		}
		if wp, ok := e.Payload["workPackageId"].(string); ok && wp != "" {
			entry.WorkPackageID = wp
		}
		entry.EventIDs = append(entry.EventIDs, e.ID)
		switch e.Type {
		case "EXECUTION_STARTED", "EXECUTION_LEASED", "EXECUTION_PLANNED":
			if entry.StartedAt == "" {
				entry.StartedAt = e.OccurredAt
			}
			if entry.Outcome == "unknown" {
				entry.Outcome = "running"
			}
		case "EXECUTION_COMPLETED":
			entry.FinishedAt = e.OccurredAt
			entry.Outcome = "succeeded"
		case "EXECUTION_FAILED":
			entry.FinishedAt = e.OccurredAt
			entry.Outcome = "failed"
		case "EXECUTION_EXPIRED":
			entry.FinishedAt = e.OccurredAt
			entry.Outcome = "expired"
		}
	}
	out := make([]JobHistoryEntry, 0, len(byID))
	for _, entry := range byID {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssignmentID < out[j].AssignmentID })
	return out, nil
}
