package store

import (
	"path/filepath"
	"testing"
)

func TestLayoutPaths(t *testing.T) {
	l := NewLayout(filepath.Join("root"))
	if l.ManifestPath() != filepath.Join("root", ManifestFile) {
		t.Errorf("ManifestPath = %q", l.ManifestPath())
	}
	if l.EventsPath() != filepath.Join("root", EventsFile) {
		t.Errorf("EventsPath = %q", l.EventsPath())
	}
	if l.RetentionPath() != filepath.Join("root", RetentionFile) {
		t.Errorf("RetentionPath = %q", l.RetentionPath())
	}
	want := filepath.Join("root", RecordsDir, "issue", "iss_1.json")
	got, err := l.RecordPath("issue", "iss_1")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("RecordPath = %q, want %q", got, want)
	}
}

func TestLayoutRejectsTraversal(t *testing.T) {
	l := NewLayout("root")
	bad := []struct{ kind, id string }{
		{"..", "x"},
		{"issue", ".."},
		{"issue", "../x"},
		{"issue", `a/b`},
		{"issue", `a\b`},
		{"a/b", "x"},
	}
	for _, c := range bad {
		if _, err := l.RecordPath(c.kind, c.id); err == nil {
			t.Errorf("expected error for kind=%q id=%q", c.kind, c.id)
		}
	}
}
