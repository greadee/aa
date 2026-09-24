package compat

import (
	"errors"
	"reflect"
	"testing"

	visualizer "github.com/greadee/aa/visualizer"
)

func TestNormalizeSnakeCase(t *testing.T) {
	m, err := NormalizePayload(map[string]any{
		"secondary_paths": []any{"b.go", "a.go", "b.go"},
		"access_sequence": []any{"read", "write"},
	})
	if err != nil {
		t.Fatalf("NormalizePayload: %v", err)
	}
	if !reflect.DeepEqual(m.SecondaryPaths, []string{"b.go", "a.go"}) {
		t.Fatalf("secondary paths = %v", m.SecondaryPaths)
	}
	if !reflect.DeepEqual(m.AccessSequence, []string{"read", "write"}) {
		t.Fatalf("access sequence = %v", m.AccessSequence)
	}
}

func TestNormalizeCamelCaseAliases(t *testing.T) {
	m, err := NormalizePayload(map[string]any{
		"secondaryPaths": []string{"x.go"},
		"accessSequence": "read",
	})
	if err != nil {
		t.Fatalf("NormalizePayload: %v", err)
	}
	if !reflect.DeepEqual(m.SecondaryPaths, []string{"x.go"}) {
		t.Fatalf("secondary paths = %v", m.SecondaryPaths)
	}
	if !reflect.DeepEqual(m.AccessSequence, []string{"read"}) {
		t.Fatalf("access sequence = %v", m.AccessSequence)
	}
}

// TestNormalizeDelimitedRealObservationMetadata mirrors the scalar form the
// obsv durable allowlist preserves over the shared transport.
func TestNormalizeDelimitedRealObservationMetadata(t *testing.T) {
	m, err := NormalizePayload(map[string]any{
		"secondary_paths": "pkg/a.go, pkg/b.go",
		"access_sequence": "read, edit, test",
		"tool":            "gopls",
		"unknown":         "ignored",
	})
	if err != nil {
		t.Fatalf("NormalizePayload: %v", err)
	}
	if !reflect.DeepEqual(m.SecondaryPaths, []string{"pkg/a.go", "pkg/b.go"}) {
		t.Fatalf("secondary paths = %v", m.SecondaryPaths)
	}
	if !reflect.DeepEqual(m.AccessSequence, []string{"read", "edit", "test"}) {
		t.Fatalf("access sequence = %v", m.AccessSequence)
	}
}

func TestNormalizeDelimitedTrimsAndDedups(t *testing.T) {
	m, err := NormalizePayload(map[string]any{
		"secondary_paths": " a.go ,, b.go , a.go ",
	})
	if err != nil {
		t.Fatalf("NormalizePayload: %v", err)
	}
	if !reflect.DeepEqual(m.SecondaryPaths, []string{"a.go", "b.go"}) {
		t.Fatalf("secondary paths = %v", m.SecondaryPaths)
	}
}

func TestNormalizeDelimitedSingleValue(t *testing.T) {
	m, err := NormalizePayload(map[string]any{"access_sequence": "read"})
	if err != nil {
		t.Fatalf("NormalizePayload: %v", err)
	}
	if !reflect.DeepEqual(m.AccessSequence, []string{"read"}) {
		t.Fatalf("access sequence = %v", m.AccessSequence)
	}
}

func TestNormalizeIgnoresUnknownMetadata(t *testing.T) {
	m, err := NormalizePayload(map[string]any{"secret": "raw prompt", "lines": 42})
	if err != nil {
		t.Fatalf("NormalizePayload: %v", err)
	}
	if !m.Empty() {
		t.Fatalf("metadata = %+v, want empty", m)
	}
}

func TestNormalizeRejectsWrongType(t *testing.T) {
	_, err := NormalizePayload(map[string]any{"secondary_paths": 42})
	if !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	_, err = NormalizePayload(map[string]any{"access_sequence": []any{"ok", 7}})
	if !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestNormalizeNilPayload(t *testing.T) {
	m, err := NormalizePayload(nil)
	if err != nil || !m.Empty() {
		t.Fatalf("m = %+v, err = %v", m, err)
	}
}

func TestNormalizeEvent(t *testing.T) {
	e := visualizer.Event{}
	e.Payload = map[string]any{"secondary_paths": []any{"a.go"}}
	m, err := Normalize(e)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(m.SecondaryPaths) != 1 {
		t.Fatalf("secondary paths = %v", m.SecondaryPaths)
	}
}

func TestKeys(t *testing.T) {
	if got := Keys(); !reflect.DeepEqual(got, []string{"secondary_paths", "access_sequence"}) {
		t.Fatalf("Keys() = %v", got)
	}
}
