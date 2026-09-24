package report

import (
	"reflect"
	"testing"

	"github.com/greadee/aa/obsv/protocol"
)

func ev(seq int, source protocol.SourceType, confidence protocol.SourceConfidence, pkg, actor string) protocol.Event {
	return protocol.Event{
		Version:       protocol.Version,
		SessionID:     "ses_1",
		Sequence:      seq,
		SourceType:    source,
		Confidence:    confidence,
		WorkPackageID: pkg,
		Actor:         actor,
	}
}

func TestBuildCountsDeterministically(t *testing.T) {
	events := []protocol.Event{
		ev(2, protocol.SourceFile, protocol.ConfidenceObserved, "wp_b", "agent"),
		ev(0, protocol.SourceTool, protocol.ConfidenceExact, "wp_a", "agent"),
		ev(1, protocol.SourceTool, protocol.ConfidenceInferred, "wp_a", "human"),
	}
	got := Build("ses_1", events)

	if got.Events != 3 || got.FirstSequence != 0 || got.LastSequence != 2 {
		t.Fatalf("report span = %d %d..%d, want 3 0..2", got.Events, got.FirstSequence, got.LastSequence)
	}
	wantTypes := []Count{{"file", 1}, {"tool", 2}}
	if !reflect.DeepEqual(got.BySourceType, wantTypes) {
		t.Fatalf("BySourceType = %#v, want %#v", got.BySourceType, wantTypes)
	}
	wantConf := []Count{{"exact", 1}, {"inferred", 1}, {"observed", 1}}
	if !reflect.DeepEqual(got.ByConfidence, wantConf) {
		t.Fatalf("ByConfidence = %#v, want %#v", got.ByConfidence, wantConf)
	}
	if !reflect.DeepEqual(got.WorkPackages, []string{"wp_a", "wp_b"}) {
		t.Fatalf("WorkPackages = %v", got.WorkPackages)
	}
	if !reflect.DeepEqual(got.Actors, []string{"agent", "human"}) {
		t.Fatalf("Actors = %v", got.Actors)
	}
}

func TestBuildIsInputOrderIndependent(t *testing.T) {
	events := []protocol.Event{
		ev(2, protocol.SourceFile, protocol.ConfidenceObserved, "wp_b", "agent"),
		ev(0, protocol.SourceTool, protocol.ConfidenceExact, "wp_a", "agent"),
		ev(1, protocol.SourceTool, protocol.ConfidenceInferred, "wp_a", "human"),
	}
	first := Build("ses_1", events)
	reversed := []protocol.Event{events[2], events[1], events[0]}
	second := Build("ses_1", reversed)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("report depends on input order:\n%#v\n%#v", first, second)
	}
}
