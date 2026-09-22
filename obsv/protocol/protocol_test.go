package protocol

import (
	"reflect"
	"strings"
	"testing"
)

func validEvent() Event {
	return Event{
		Version:    Version,
		SessionID:  "ses_1",
		Sequence:   0,
		SourceType: SourceTool,
		Confidence: ConfidenceObserved,
	}
}

func TestValidateAcceptsMinimalEvent(t *testing.T) {
	if err := validEvent().Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Event){
		"missing version": func(e *Event) { e.Version = "" },
		"wrong version":   func(e *Event) { e.Version = "0.9" },
		"missing session": func(e *Event) { e.SessionID = " " },
		"negative seq":    func(e *Event) { e.Sequence = -1 },
		"bad source":      func(e *Event) { e.SourceType = "network" },
		"bad confidence":  func(e *Event) { e.Confidence = "certain" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEvent()
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatalf("Validate() = nil, want error")
			}
		})
	}
}

func TestConfidenceIsNeverPromotable(t *testing.T) {
	for _, c := range Confidences() {
		if c.Promotable() {
			t.Fatalf("%s.Promotable() = true, want false", c)
		}
	}
}

func TestSanitizeKeepsAllowlistAndDropsDenied(t *testing.T) {
	attrs := map[string]any{
		"tool":    "gopls",
		"command": "go test ./...",
		"path":    "obsv/protocol",
		"count":   3,
		"prompt":  "do the thing",
		"content": "secret source",
		"token":   "abc123",
		"unknown": "nope",
	}
	got := Sanitize(attrs)
	want := map[string]string{
		"tool":    "gopls",
		"command": "go test ./...",
		"path":    "obsv/protocol",
		"count":   "3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sanitize() = %#v, want %#v", got, want)
	}
}

func TestSanitizeKeepsVisualizerProfileKeys(t *testing.T) {
	attrs := map[string]any{
		"secondary_paths": "pkg/a.go, pkg/b.go",
		"access_sequence": "read, edit, test",
		"tool":            "gopls",
	}
	got := Sanitize(attrs)
	want := map[string]string{
		"secondary_paths": "pkg/a.go, pkg/b.go",
		"access_sequence": "read, edit, test",
		"tool":            "gopls",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sanitize() = %#v, want %#v", got, want)
	}
}

func TestMetadataKeysIncludeVisualizerProfileKeys(t *testing.T) {
	keys := MetadataKeys()
	have := map[string]bool{}
	for _, k := range keys {
		have[k] = true
	}
	for _, want := range []string{"secondary_paths", "access_sequence"} {
		if !have[want] {
			t.Fatalf("MetadataKeys() missing %q: %v", want, keys)
		}
	}
}

func TestSanitizeDropsNonScalars(t *testing.T) {
	got := Sanitize(map[string]any{
		"tool":    []string{"a", "b"},
		"command": map[string]any{"x": 1},
		"path":    "ok",
	})
	want := map[string]string{"path": "ok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sanitize() = %#v, want %#v", got, want)
	}
}

func TestSanitizeTruncatesLongValues(t *testing.T) {
	long := strings.Repeat("x", MaxValueLen+50)
	got := Sanitize(map[string]any{"command": long})
	if len([]rune(got["command"])) != MaxValueLen {
		t.Fatalf("len = %d, want %d", len([]rune(got["command"])), MaxValueLen)
	}
}

func TestSanitizeIsDeterministic(t *testing.T) {
	attrs := map[string]any{"tool": "a", "path": "b", "count": 1, "exitCode": 0}
	first := Sanitize(attrs)
	for i := 0; i < 20; i++ {
		if !reflect.DeepEqual(first, Sanitize(attrs)) {
			t.Fatal("Sanitize is not deterministic")
		}
	}
}

func TestSanitizeEmptyReturnsNil(t *testing.T) {
	if got := Sanitize(nil); got != nil {
		t.Fatalf("Sanitize(nil) = %#v, want nil", got)
	}
	if got := Sanitize(map[string]any{"prompt": "x"}); got != nil {
		t.Fatalf("Sanitize(denied only) = %#v, want nil", got)
	}
}

func TestMetadataKeysAreSorted(t *testing.T) {
	keys := MetadataKeys()
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Fatalf("keys not strictly sorted: %v", keys)
		}
	}
}
