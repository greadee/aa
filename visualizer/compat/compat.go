// Package compat is the visualizer's explicit observation-metadata profile.
//
// Adopting observation metadata is deliberate and versioned: only the keys
// named here are normalized, values are validated, and unknown metadata is
// ignored. This keeps the visualizer independent of any single observation
// producer while remaining testable.
//
// The shared obsv transport keeps only scalar metadata, so the profile keys
// arrive over the transport as comma-separated strings; the profile normalizes
// both that real form and the already-split list form a native contract payload
// may carry. Version 1.1 adds delimited-scalar normalization.
package compat

import (
	"fmt"
	"strings"

	visualizer "github.com/greadee/aa/visualizer"
)

// Version is the profile version.
const Version = "1.1"

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
		return splitDelimited(v), nil
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

// splitDelimited normalizes a scalar observation-metadata value into its
// elements. The obsv durable allowlist keeps only scalar values, so the profile
// keys arrive over the shared transport as comma-separated strings; a single
// undelimited value stays a one-element list. Empty elements are dropped.
func splitDelimited(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
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
