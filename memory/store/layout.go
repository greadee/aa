package store

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Portable layout file and directory names.
const (
	ManifestFile  = "manifest.json"
	EventsFile    = "events.jsonl"
	RetentionFile = "retention.json"
	RecordsDir    = "records"
)

// Layout describes the portable project layout rooted at a directory.
type Layout struct {
	Root string
}

// NewLayout returns the layout rooted at root.
func NewLayout(root string) Layout {
	return Layout{Root: root}
}

// ManifestPath returns the canonical project manifest path.
func (l Layout) ManifestPath() string { return filepath.Join(l.Root, ManifestFile) }

// EventsPath returns the append-only event log path.
func (l Layout) EventsPath() string { return filepath.Join(l.Root, EventsFile) }

// RetentionPath returns the retention floor path.
func (l Layout) RetentionPath() string { return filepath.Join(l.Root, RetentionFile) }

// RecordsRoot returns the directory holding materialized records.
func (l Layout) RecordsRoot() string { return filepath.Join(l.Root, RecordsDir) }

// KindDir returns the directory for a record kind.
func (l Layout) KindDir(kind string) (string, error) {
	if err := safeSegment("kind", kind); err != nil {
		return "", err
	}
	return filepath.Join(l.RecordsRoot(), kind), nil
}

// RecordPath returns the path for a record, rejecting path traversal.
func (l Layout) RecordPath(kind, id string) (string, error) {
	dir, err := l.KindDir(kind)
	if err != nil {
		return "", err
	}
	if err := safeSegment("id", id); err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

var segmentRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:@+-]*$`)

// safeSegment rejects empty, traversal, and separator-bearing path segments.
func safeSegment(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s: is required", field)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s: invalid segments %q", field, value)
	}
	if strings.ContainsAny(value, `/\`) || strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s: path separators are not allowed", field)
	}
	if !segmentRE.MatchString(value) {
		return fmt.Errorf("%s: invalid segment %q", field, value)
	}
	return nil
}
