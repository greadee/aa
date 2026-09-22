package codex

import (
	"reflect"
	"strings"
	"testing"

	"github.com/greadee/aa/obsv/protocol"
)

func TestTranslateToolLine(t *testing.T) {
	line := []byte(`{"type":"tool.completed","session_id":"ses_9","timestamp":"2026-01-01T00:00:00Z","tool":"gopls","durationMs":12,"command":"go test ./...","prompt":"leak"}`)
	ev, ok, err := TranslateLine("ses_1", 0, line)
	if err != nil || !ok {
		t.Fatalf("TranslateLine() = %v, %v", ok, err)
	}
	if ev.SessionID != "ses_9" {
		t.Fatalf("SessionID = %q, want ses_9", ev.SessionID)
	}
	if ev.SourceType != protocol.SourceTool {
		t.Fatalf("SourceType = %q, want tool", ev.SourceType)
	}
	if ev.Confidence != protocol.ConfidenceObserved {
		t.Fatalf("Confidence = %q, want observed", ev.Confidence)
	}
	want := map[string]string{"tool": "gopls", "durationMs": "12", "command": "go test ./..."}
	if !reflect.DeepEqual(ev.Metadata, want) {
		t.Fatalf("Metadata = %#v, want %#v", ev.Metadata, want)
	}
}

func TestTranslateFileAndSessionTypes(t *testing.T) {
	cases := map[string]protocol.SourceType{
		`{"type":"session.started"}`:               protocol.SourceSession,
		`{"type":"file.patch","path":"a.go"}`:      protocol.SourceFile,
		`{"type":"turn.completed"}`:                protocol.SourceWorkDelta,
		`{"msg":{"type":"tool.call"},"tool":"rg"}`: protocol.SourceTool,
	}
	for line, want := range cases {
		ev, ok, err := TranslateLine("ses_1", 0, []byte(line))
		if err != nil || !ok {
			t.Fatalf("TranslateLine(%s) = %v, %v", line, ok, err)
		}
		if ev.SourceType != want {
			t.Fatalf("%s: SourceType = %q, want %q", line, ev.SourceType, want)
		}
	}
}

func TestTranslateConfidenceOverride(t *testing.T) {
	ev, ok, err := TranslateLine("ses_1", 0, []byte(`{"type":"file.read","source_confidence":"exact"}`))
	if err != nil || !ok {
		t.Fatalf("TranslateLine() = %v, %v", ok, err)
	}
	if ev.Confidence != protocol.ConfidenceExact {
		t.Fatalf("Confidence = %q, want exact", ev.Confidence)
	}
}

func TestTranslateBlankAndUntypedLines(t *testing.T) {
	if _, ok, err := TranslateLine("ses_1", 0, []byte("   ")); ok || err != nil {
		t.Fatalf("blank = %v, %v, want false, nil", ok, err)
	}
	if _, ok, err := TranslateLine("ses_1", 0, []byte(`{"foo":"bar"}`)); ok || err != nil {
		t.Fatalf("untyped = %v, %v, want false, nil", ok, err)
	}
}

func TestTranslateInvalidJSON(t *testing.T) {
	if _, _, err := TranslateLine("ses_1", 0, []byte(`{not json`)); err == nil {
		t.Fatal("TranslateLine(invalid) = nil, want error")
	}
}

func TestTranslateReaderAssignsSequence(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"session.started"}`,
		``,
		`{"type":"tool.call","tool":"rg"}`,
		`{"type":"session.completed"}`,
	}, "\n")
	events, err := Translate("ses_1", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Translate() = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len = %d, want 3", len(events))
	}
	for i, ev := range events {
		if ev.Sequence != i {
			t.Fatalf("sequence[%d] = %d, want %d", i, ev.Sequence, i)
		}
	}
}

func TestTranslateIsDeterministic(t *testing.T) {
	line := []byte(`{"type":"tool.call","tool":"rg","path":"x"}`)
	first, _, _ := TranslateLine("ses_1", 0, line)
	for i := 0; i < 20; i++ {
		got, _, _ := TranslateLine("ses_1", 0, line)
		if !reflect.DeepEqual(first, got) {
			t.Fatal("translation is not deterministic")
		}
	}
}
