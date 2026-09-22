// Package codex translates Codex `exec --json` JSONL output into aa-obsv
// observation events.
//
// The translator is deliberately tolerant: each line is a JSON object and the
// event type is read from `type`, falling back to a nested `msg.type` or
// `item.type`. Only allowlisted scalar fields survive as metadata; nested
// objects and content-bearing keys are dropped by the protocol sanitizer. The
// translator is deterministic and assigns sequence locally.
package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/greadee/aa/obsv/protocol"
)

// TranslateLine converts one JSONL line into an observation event. It returns
// ok=false for blank or untyped lines that carry no observation.
func TranslateLine(sessionID string, sequence int, line []byte) (protocol.Event, bool, error) {
	if len(strings.TrimSpace(string(line))) == 0 {
		return protocol.Event{}, false, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return protocol.Event{}, false, fmt.Errorf("obsv/codex: decode line: %w", err)
	}
	kind := firstString(raw, "type", "event_type", "eventType")
	if kind == "" {
		kind = nestedType(raw)
	}
	if kind == "" {
		return protocol.Event{}, false, nil
	}

	ev := protocol.Event{
		Version:       protocol.Version,
		SessionID:     sessionID,
		Sequence:      sequence,
		OccurredAt:    firstString(raw, "timestamp", "occurred_at", "occurredAt", "created_at"),
		SourceType:    sourceType(kind),
		Action:        kind,
		Confidence:    confidence(raw),
		Actor:         firstString(raw, "actor", "role"),
		WorkPackageID: firstString(raw, "work_package_id", "workPackageId", "package_id"),
		AttemptID:     firstString(raw, "attempt_id", "attemptId"),
	}
	if sid := firstString(raw, "session_id", "sessionId"); sid != "" {
		ev.SessionID = sid
	}
	ev.Metadata = protocol.Sanitize(raw)
	if err := ev.Validate(); err != nil {
		return protocol.Event{}, false, err
	}
	return ev, true, nil
}

// Translate reads JSONL from r and returns the observation events in order,
// assigning sequences from zero.
func Translate(sessionID string, r io.Reader) ([]protocol.Event, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var events []protocol.Event
	sequence := 0
	for scanner.Scan() {
		ev, ok, err := TranslateLine(sessionID, sequence, scanner.Bytes())
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		events = append(events, ev)
		sequence++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("obsv/codex: read: %w", err)
	}
	return events, nil
}

func sourceType(kind string) protocol.SourceType {
	k := strings.ToLower(kind)
	switch {
	case strings.Contains(k, "session"):
		return protocol.SourceSession
	case strings.Contains(k, "tool"), strings.Contains(k, "command"), strings.Contains(k, "cmd"), strings.Contains(k, "exec"):
		return protocol.SourceTool
	case strings.Contains(k, "file"), strings.Contains(k, "patch"), strings.Contains(k, "edit"), strings.Contains(k, "read"), strings.Contains(k, "write"):
		return protocol.SourceFile
	default:
		return protocol.SourceWorkDelta
	}
}

func confidence(raw map[string]any) protocol.SourceConfidence {
	if v, ok := raw["source_confidence"].(string); ok {
		c := protocol.SourceConfidence(v)
		if protocol.ValidConfidence(c) {
			return c
		}
	}
	return protocol.ConfidenceObserved
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := raw[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func nestedType(raw map[string]any) string {
	for _, key := range []string{"msg", "item", "payload"} {
		if nested, ok := raw[key].(map[string]any); ok {
			if t, ok := nested["type"].(string); ok && t != "" {
				return t
			}
		}
	}
	return ""
}
