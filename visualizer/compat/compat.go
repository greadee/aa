// Package compat is the visualizer's explicit observation-metadata profile.
//
// Adopting observation metadata is deliberate and versioned: only the keys
// named here are normalized, values are validated, and unknown metadata is
// ignored. This keeps the visualizer independent of any single observation
// producer while remaining testable.
package compat

import (
	"fmt"

	visualizer "github.com/greadee/aa/visualizer"
)

// Version is the profile version.
const Version = "1.0"

// Canonical metadata keys and their accepted aliases.
const (
	KeySecondaryPaths = "secondary_paths"
	KeyAccessSequence = "access_sequence"
)

var secondaryPathKeys = []string{KeySecondaryPaths, "secondaryPaths"}
var accessSequenceKeys = []string{KeyAccessSequence, "accessSequence"}

// Metadata is the normalized observation metadata for one event.
type Metadata struct {
	SecondaryPaths []string `json:"secondaryPaths,omitempty"`
	AccessSequence []string `json:"accessSequence,omitempty"`
}

// Empty reports whether the metadata carries nothing.
func (m Metadata) Empty() bool {
	return len(m.SecondaryPaths) == 0 && len(m.AccessSequence) == 0
}

// Keys returns the canonical supported keys.
func Keys() []string {
	return []string{KeySecondaryPaths, KeyAccessSequence}
}

// Normalize extracts and validates the supported metadata from an event.
func Normalize(e visualizer.Event) (Metadata, error) {
	return NormalizePayload(e.Payload)
}

// NormalizePayload extracts and validates the supported metadata from a payload.
//
// It returns an error only when a supported key is present with an invalid
// value; unknown keys are ignored.
func NormalizePayload(payload map[string]any) (Metadata, error) {
	if payload == nil {
		return Metadata{}, nil
	}
	var m Metadata
	if raw, ok := lookup(payload, secondaryPathKeys); ok {
		values, err := toStrings(KeySecondaryPaths, raw)
		if err != nil {
			return Metadata{}, err
		}
		m.SecondaryPaths = dedup(values)
	}
	if raw, ok := lookup(payload, accessSequenceKeys); ok {
		values, err := toStrings(KeyAccessSequence, raw)
		if err != nil {
			return Metadata{}, err
		}
		m.AccessSequence = dedup(values)
	}
	return m, nil
}

func lookup(payload map[string]any, keys []string) (any, bool) {
	for _, key := range keys {
		if v, ok := payload[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func toStrings(field string, value any) ([]string, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case string:
		if v == "" {
			return nil, nil
		}
		return []string{v}, nil
	case []string:
		return append([]string(nil), v...), nil
	case []any:
		out := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%w: %s[%d] must be a string", visualizer.ErrInvalid, field, i)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w: %s must be a string or list of strings", visualizer.ErrInvalid, field)
	}
}

func dedup(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
