package v1

import "fmt"

// EventType enumerates canonical orchestration and execution events.
// Observation events are owned by aa-obsv and are not listed here.
type EventType string

// Canonical event types.
const (
	EventProjectRegistered     EventType = "PROJECT_REGISTERED"
	EventProjectUpdated        EventType = "PROJECT_UPDATED"
	EventWorkPackageCreated    EventType = "WORK_PACKAGE_CREATED"
	EventWorkPackageReady      EventType = "WORK_PACKAGE_READY"
	EventWorkPackageCompleted  EventType = "WORK_PACKAGE_COMPLETED"
	EventDependencyGraphRevise EventType = "DEPENDENCY_GRAPH_REVISED"
	EventExecutionPlanned      EventType = "EXECUTION_PLANNED"
	EventExecutionLeased       EventType = "EXECUTION_LEASED"
	EventExecutionStarted      EventType = "EXECUTION_STARTED"
	EventExecutionPaused       EventType = "EXECUTION_PAUSED"
	EventExecutionResumed      EventType = "EXECUTION_RESUMED"
	EventExecutionCompleted    EventType = "EXECUTION_COMPLETED"
	EventExecutionFailed       EventType = "EXECUTION_FAILED"
	EventExecutionExpired      EventType = "EXECUTION_EXPIRED"
	EventResultRecorded        EventType = "RESULT_RECORDED"
	EventTestRecorded          EventType = "TEST_RECORDED"
	EventReviewRecorded        EventType = "REVIEW_RECORDED"
	EventHandoffCreated        EventType = "HANDOFF_CREATED"
	EventArtifactRecorded      EventType = "ARTIFACT_RECORDED"
	EventWorkAccepted          EventType = "WORK_ACCEPTED"
	EventWorkRejected          EventType = "WORK_REJECTED"
	EventDeficiencyRaised      EventType = "DEFICIENCY_RAISED"
	EventTelemetryRecorded     EventType = "TELEMETRY_RECORDED"
	EventMemoryCandidateCrtd   EventType = "MEMORY_CANDIDATE_CREATED"
	EventMemoryValidated       EventType = "MEMORY_VALIDATED"
	EventMemoryPromoted        EventType = "MEMORY_PROMOTED"
	EventMemorySuperseded      EventType = "MEMORY_SUPERSEDED"
	EventMemoryArchived        EventType = "MEMORY_ARCHIVED"
)

var eventTypes = map[EventType]bool{
	EventProjectRegistered: true, EventProjectUpdated: true,
	EventWorkPackageCreated: true, EventWorkPackageReady: true, EventWorkPackageCompleted: true,
	EventDependencyGraphRevise: true,
	EventExecutionPlanned:      true, EventExecutionLeased: true, EventExecutionStarted: true,
	EventExecutionPaused: true, EventExecutionResumed: true, EventExecutionCompleted: true,
	EventExecutionFailed: true, EventExecutionExpired: true,
	EventResultRecorded: true, EventTestRecorded: true, EventReviewRecorded: true,
	EventHandoffCreated: true, EventArtifactRecorded: true,
	EventWorkAccepted: true, EventWorkRejected: true, EventDeficiencyRaised: true,
	EventTelemetryRecorded:   true,
	EventMemoryCandidateCrtd: true, EventMemoryValidated: true, EventMemoryPromoted: true,
	EventMemorySuperseded: true, EventMemoryArchived: true,
}

// EventAggregate identifies the entity an event applies to.
type EventAggregate struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version *int   `json:"version,omitempty"`
}

// Event is a canonical orchestration or execution event.
type Event struct {
	Envelope
	Sequence      int            `json:"sequence"`
	OccurredAt    string         `json:"occurredAt"`
	RecordedAt    string         `json:"recordedAt,omitempty"`
	Type          EventType      `json:"type"`
	Aggregate     EventAggregate `json:"aggregate"`
	Actor         Actor          `json:"actor"`
	CausationID   string         `json:"causationId,omitempty"`
	CorrelationID string         `json:"correlationId,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
}

// Validate checks the event contract.
func (e Event) Validate() error {
	if e.Kind != "event" {
		return fmt.Errorf("kind: expected %q, got %q", "event", e.Kind)
	}
	if err := e.Envelope.Validate(); err != nil {
		return err
	}
	if e.Sequence < 0 {
		return fmt.Errorf("sequence: must be >= 0")
	}
	if err := RequireTimestamp("occurredAt", e.OccurredAt); err != nil {
		return err
	}
	if err := OptionalTimestamp("recordedAt", e.RecordedAt); err != nil {
		return err
	}
	if !eventTypes[e.Type] {
		return fmt.Errorf("type: unknown event type %q", e.Type)
	}
	if err := RequireEnum("aggregate.kind", e.Aggregate.Kind,
		"project", "work_package", "assignment", "attempt", "artifact",
		"issue", "strategy", "memory", "tool", "workflow"); err != nil {
		return err
	}
	if err := RequireIdentifier("aggregate.id", e.Aggregate.ID); err != nil {
		return err
	}
	if err := e.Actor.Validate(); err != nil {
		return err
	}
	if err := OptionalIdentifier("causationId", e.CausationID); err != nil {
		return err
	}
	return OptionalIdentifier("correlationId", e.CorrelationID)
}
