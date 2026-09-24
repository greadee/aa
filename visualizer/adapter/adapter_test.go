package adapter

import (
	"errors"
	"reflect"
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/obsv/protocol"
	visualizer "github.com/greadee/aa/visualizer"
)

func obsEvent() protocol.Event {
	return protocol.Event{
		Version:       protocol.Version,
		SessionID:     "sess_1",
		Sequence:      7,
		OccurredAt:    "2026-01-01T00:00:07Z",
		SourceType:    protocol.SourceTool,
		Source:        "bash",
		Action:        "run",
		Confidence:    protocol.ConfidenceObserved,
		Actor:         "worker_1",
		WorkPackageID: "wp_1",
		AttemptID:     "att_1",
		Metadata: map[string]string{
			"command":         "go test",
			"secondary_paths": "b.go",
			"access_sequence": "read",
			"unknown":         "ignored by the profile",
		},
	}
}

func TestToContractMapsFields(t *testing.T) {
	out, err := ToContract(obsEvent())
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	if out.Kind != "event" || out.ContractVersion != v1.Version {
		t.Fatalf("envelope = %q %q", out.Kind, out.ContractVersion)
	}
	if out.ID != "sess_1:7" {
		t.Fatalf("id = %q", out.ID)
	}
	if out.Sequence != 7 || out.OccurredAt != "2026-01-01T00:00:07Z" {
		t.Fatalf("sequence/occurredAt = %d %q", out.Sequence, out.OccurredAt)
	}
	if out.Type != v1.EventTelemetryRecorded {
		t.Fatalf("type = %q", out.Type)
	}
	if out.Aggregate.Kind != "work_package" || out.Aggregate.ID != "wp_1" {
		t.Fatalf("aggregate = %+v", out.Aggregate)
	}
	if out.Actor.Kind != "agent" || out.Actor.ID != "worker_1" {
		t.Fatalf("actor = %+v", out.Actor)
	}
	for key, want := range map[string]any{
		"sessionId":        "sess_1",
		"sequence":         7,
		"sourceType":       "tool",
		"sourceConfidence": "observed",
		"source":           "bash",
		"action":           "run",
		"workPackageId":    "wp_1",
		"attemptId":        "att_1",
		"secondary_paths":  "b.go",
		"access_sequence":  "read",
		"unknown":          "ignored by the profile",
	} {
		if got := out.Payload[key]; !reflect.DeepEqual(got, want) {
			t.Fatalf("payload[%q] = %#v, want %#v", key, got, want)
		}
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("mapped event does not validate against contracts: %v", err)
	}
}

func TestToContractAttemptAggregate(t *testing.T) {
	ev := obsEvent()
	ev.WorkPackageID = ""
	out, err := ToContract(ev)
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	if out.Aggregate.Kind != "attempt" || out.Aggregate.ID != "att_1" {
		t.Fatalf("aggregate = %+v", out.Aggregate)
	}
	if _, ok := out.Payload["workPackageId"]; ok {
		t.Fatalf("payload carried a work package without one: %#v", out.Payload)
	}
	if out.Payload["attemptId"] != "att_1" {
		t.Fatalf("attemptId = %#v", out.Payload["attemptId"])
	}
}

func TestToContractSourceTypeAggregates(t *testing.T) {
	cases := []struct {
		source protocol.SourceType
		kind   string
	}{
		{protocol.SourceSession, "workflow"},
		{protocol.SourceTool, "tool"},
		{protocol.SourceFile, "artifact"},
		{protocol.SourceWorkDelta, "work_package"},
	}
	for _, tc := range cases {
		ev := obsEvent()
		ev.WorkPackageID = ""
		ev.AttemptID = ""
		ev.SourceType = tc.source
		out, err := ToContract(ev)
		if err != nil {
			t.Fatalf("%s: ToContract: %v", tc.source, err)
		}
		if out.Aggregate.Kind != tc.kind {
			t.Fatalf("%s: aggregate kind = %q, want %q", tc.source, out.Aggregate.Kind, tc.kind)
		}
	}
}

func TestToContractActorFallback(t *testing.T) {
	ev := obsEvent()
	ev.Actor = ""
	out, err := ToContract(ev)
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	if out.Actor.Kind != "system" || out.Actor.ID != "obsv" {
		t.Fatalf("actor = %+v", out.Actor)
	}
}

func TestToContractEmptyOccurredAtFallsBack(t *testing.T) {
	ev := obsEvent()
	ev.OccurredAt = ""
	out, err := ToContract(ev)
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	if out.OccurredAt != zeroTimestamp {
		t.Fatalf("occurredAt = %q, want %q", out.OccurredAt, zeroTimestamp)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("fallback event does not validate: %v", err)
	}
}

func TestToContractNormalizesIdentity(t *testing.T) {
	ev := obsEvent()
	ev.SessionID = "sess/1"
	ev.WorkPackageID = "wp 1/2"
	out, err := ToContract(ev)
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	if out.Aggregate.ID != "wp-1-2" {
		t.Fatalf("aggregate id = %q", out.Aggregate.ID)
	}
	if out.Payload["workPackageId"] != "wp-1-2" {
		t.Fatalf("payload workPackageId = %#v", out.Payload["workPackageId"])
	}
	if out.ID != "sess-1:7" {
		t.Fatalf("id = %q", out.ID)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("normalized event does not validate: %v", err)
	}
}

func TestToContractRejectsInvalidObservation(t *testing.T) {
	ev := obsEvent()
	ev.Confidence = "certain"
	if _, err := ToContract(ev); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestToContractIsDeterministic(t *testing.T) {
	first, err := ToContract(obsEvent())
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	second, err := ToContract(obsEvent())
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("adapter is not deterministic:\n%+v\n%+v", first, second)
	}
}

func TestToContractOutputIsAValidContractEvent(t *testing.T) {
	out, err := ToContract(obsEvent())
	if err != nil {
		t.Fatalf("ToContract: %v", err)
	}
	// The mapped event must satisfy every rule the shared taxonomy enforces so
	// the projection can treat it exactly like a native contract event.
	var contract visualizer.Event = out
	if err := contract.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
