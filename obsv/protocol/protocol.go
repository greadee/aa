// Package protocol defines aa-obsv's observation protocol v1.
//
// Observation events are owned by aa-obsv, not by aa-contracts: the contracts
// event taxonomy explicitly defers observation to this module. The protocol is
// metadata-only. Callers pass arbitrary attributes and Sanitize keeps only the
// durable allowlist, so content, prompts, source, and credentials never reach a
// journal. Confidence is semantic (exact, correlated, observed, inferred) and is
// never promoted.
package protocol

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Version is the observation protocol version.
const Version = "1.0"

// Bounds applied by Sanitize.
const (
	// MaxMetadataKeys is the maximum number of metadata keys kept.
	MaxMetadataKeys = 32
	// MaxValueLen is the maximum length, in runes, of a metadata value.
	MaxValueLen = 256
)

// SourceType is the kind of work an observation describes.
type SourceType string

// Observation source types.
const (
	SourceSession   SourceType = "session"
	SourceTool      SourceType = "tool"
	SourceFile      SourceType = "file"
	SourceWorkDelta SourceType = "work_delta"
)

// SourceConfidence records how an observation was obtained.
type SourceConfidence string

// Observation confidence levels, ordered from strongest to weakest.
const (
	ConfidenceExact      SourceConfidence = "exact"
	ConfidenceCorrelated SourceConfidence = "correlated"
	ConfidenceObserved   SourceConfidence = "observed"
	ConfidenceInferred   SourceConfidence = "inferred"
)

// Event is one observation of work.
type Event struct {
	Version       string            `json:"version"`
	SessionID     string            `json:"sessionId"`
	Sequence      int               `json:"sequence"`
	OccurredAt    string            `json:"occurredAt,omitempty"`
	SourceType    SourceType        `json:"sourceType"`
	Source        string            `json:"source,omitempty"`
	Action        string            `json:"action,omitempty"`
	Confidence    SourceConfidence  `json:"sourceConfidence"`
	Actor         string            `json:"actor,omitempty"`
	WorkPackageID string            `json:"workPackageId,omitempty"`
	AttemptID     string            `json:"attemptId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// SourceTypes returns the supported source types in stable order.
func SourceTypes() []SourceType {
	return []SourceType{SourceSession, SourceTool, SourceFile, SourceWorkDelta}
}

// Confidences returns the supported confidence levels in strength order.
func Confidences() []SourceConfidence {
	return []SourceConfidence{ConfidenceExact, ConfidenceCorrelated, ConfidenceObserved, ConfidenceInferred}
}

// ValidSourceType reports whether t is a supported source type.
func ValidSourceType(t SourceType) bool {
	switch t {
	case SourceSession, SourceTool, SourceFile, SourceWorkDelta:
		return true
	default:
		return false
	}
}

// ValidConfidence reports whether c is a supported confidence level.
func ValidConfidence(c SourceConfidence) bool {
	switch c {
	case ConfidenceExact, ConfidenceCorrelated, ConfidenceObserved, ConfidenceInferred:
		return true
	default:
		return false
	}
}

// Promotable always reports false: observation confidence is semantic and is
// never promoted into canonical knowledge.
func (c SourceConfidence) Promotable() bool { return false }

// Validate checks an observation event.
func (e Event) Validate() error {
	if e.Version == "" {
		return fmt.Errorf("obsv/protocol: version is required")
	}
	if e.Version != Version {
		return fmt.Errorf("obsv/protocol: unsupported version %q", e.Version)
	}
	if strings.TrimSpace(e.SessionID) == "" {
		return fmt.Errorf("obsv/protocol: sessionId is required")
	}
	if e.Sequence < 0 {
		return fmt.Errorf("obsv/protocol: sequence must be >= 0")
	}
	if !ValidSourceType(e.SourceType) {
		return fmt.Errorf("obsv/protocol: unknown sourceType %q", e.SourceType)
	}
	if !ValidConfidence(e.Confidence) {
		return fmt.Errorf("obsv/protocol: unknown sourceConfidence %q", e.Confidence)
	}
	return nil
}

// allowlist is the durable set of metadata keys observation may carry. Keys not
// listed here are dropped by Sanitize.
//
// secondary_paths and access_sequence are the visualizer observation-metadata
// profile keys (visualizer/compat). They are observation metadata, not content,
// and must survive sanitization so the profile can normalize them over the
// shared obsv transport (ISS-OBSV-2).
var allowlist = map[string]bool{
	"tool":            true,
	"command":         true,
	"operation":       true,
	"target":          true,
	"direction":       true,
	"path":            true,
	"language":        true,
	"branch":          true,
	"commit":          true,
	"repo":            true,
	"status":          true,
	"outcome":         true,
	"testName":        true,
	"model":           true,
	"provider":        true,
	"kind":            true,
	"count":           true,
	"bytes":           true,
	"line":            true,
	"column":          true,
	"exitCode":        true,
	"durationMs":      true,
	"url":             true,
	"secondary_paths": true,
	"access_sequence": true,
}

// denied keys must never survive sanitization even if a caller passes them.
var denied = map[string]bool{
	"content": true, "prompt": true, "source": true, "code": true,
	"diff": true, "patch": true, "text": true, "body": true, "message": true,
	"token": true, "secret": true, "password": true, "credential": true,
	"authorization": true, "apikey": true, "apiKey": true, "env": true,
}

// MetadataKeys returns the allowlisted metadata keys in stable order.
func MetadataKeys() []string {
	keys := make([]string, 0, len(allowlist))
	for k := range allowlist {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Sanitize keeps only allowlisted, non-denied, scalar metadata, coerced to a
// bounded string. It is deterministic: the same attributes always produce the
// same result. Returns nil when nothing survives.
func Sanitize(attrs map[string]any) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]string)
	for _, k := range keys {
		if len(out) >= MaxMetadataKeys {
			break
		}
		if denied[k] || !allowlist[k] {
			continue
		}
		v, ok := scalar(attrs[k])
		if !ok {
			continue
		}
		v = truncate(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SanitizeMetadata applies the allowlist to already-stringified metadata.
func SanitizeMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	attrs := make(map[string]any, len(in))
	for k, v := range in {
		attrs[k] = v
	}
	return Sanitize(attrs)
}

func scalar(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", false
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case int:
		return strconv.Itoa(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), true
		}
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 32), true
	default:
		return "", false
	}
}

func truncate(v string) string {
	if utf8.RuneCountInString(v) <= MaxValueLen {
		return v
	}
	runes := []rune(v)
	return string(runes[:MaxValueLen])
}
