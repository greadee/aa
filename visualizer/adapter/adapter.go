// Package adapter is the visualizer's single mapping from the shared aa-obsv
// observation protocol to the visualizer event type.
//
// The visualizer defines no observation protocol of its own: it consumes the
// contracts event taxonomy through one explicit adapter so that live and replay
// observe the same vocabulary. The mapping is pure and deterministic: the same
// observation always yields the same contract event. Session, sequence, actor,
// work-package/attempt identity, and observation metadata are mapped here, and
// unknown metadata is carried through untouched for the compatibility profile
// to ignore.
package adapter

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/obsv/protocol"
	visualizer "github.com/greadee/aa/visualizer"
)

// contractKind is the fixed envelope kind of the shared event taxonomy.
const contractKind = "event"

// zeroTimestamp is the deterministic fallback for an observation without a
// timestamp. Ordering always derives from Sequence, never this value.
const zeroTimestamp = "1970-01-01T00:00:00Z"

// ToContract maps one aa-obsv observation event to the visualizer event type
// (the contracts v1.Event taxonomy).
//
// Every observation is recorded as a TELEMETRY_RECORDED event. The aggregate is
// derived from the observation's work identity, preferring a work package, then
// an attempt, and falling back to the observation source. The original
// session/sequence and the work-package/attempt identity are also placed in the
// payload, together with the sanitized observation metadata, so the projection
// and the compatibility profile see the same fields a native contract event
// carries.
func ToContract(ev protocol.Event) (visualizer.Event, error) {
	if err := ev.Validate(); err != nil {
		return visualizer.Event{}, fmt.Errorf("%w: %s", visualizer.ErrInvalid, err)
	}

	aggregateKind, aggregateID, workPackageID, attemptID := identity(ev)

	payload := make(map[string]any, len(ev.Metadata)+8)
	for k, v := range ev.Metadata {
		payload[k] = v
	}
	payload["sessionId"] = ev.SessionID
	payload["sequence"] = ev.Sequence
	payload["sourceType"] = string(ev.SourceType)
	payload["sourceConfidence"] = string(ev.Confidence)
	if ev.Source != "" {
		payload["source"] = ev.Source
	}
	if ev.Action != "" {
		payload["action"] = ev.Action
	}
	if workPackageID != "" {
		payload["workPackageId"] = workPackageID
	}
	if attemptID != "" {
		payload["attemptId"] = attemptID
	}

	out := visualizer.Event{}
	out.ContractVersion = v1.Version
	out.Kind = contractKind
	out.ID = observationID(ev)
	out.Sequence = ev.Sequence
	out.OccurredAt = occurredAt(ev)
	out.Type = v1.EventTelemetryRecorded
	out.Aggregate = visualizer.EventAggregate{Kind: aggregateKind, ID: aggregateID}
	out.Actor = actor(ev)
	out.Payload = payload

	if err := out.Validate(); err != nil {
		return visualizer.Event{}, fmt.Errorf("%w: %s", visualizer.ErrInvalid, err)
	}
	return out, nil
}

// identity derives the aggregate kind/id and the work-package/attempt identity.
// A work package takes precedence over an attempt, which takes precedence over
// the observation source. Values are normalized into valid contract identifiers
// so the mapping never depends on a caller's id spelling.
func identity(ev protocol.Event) (kind, id, workPackageID, attemptID string) {
	if ev.WorkPackageID != "" {
		workPackageID = identifier(ev.WorkPackageID)
		return "work_package", workPackageID, workPackageID, identifier(ev.AttemptID)
	}
	if ev.AttemptID != "" {
		attemptID = identifier(ev.AttemptID)
		return "attempt", attemptID, "", attemptID
	}
	return aggregateKind(ev.SourceType), identifier(firstNonEmpty(ev.Source, ev.Action, ev.SessionID)), "", ""
}

// aggregateKind maps an observation source type onto the shared aggregate kind.
// A session observation is a workflow: the contract taxonomy reserves the
// session node for the projection, which derives it from the session id.
func aggregateKind(st protocol.SourceType) string {
	switch st {
	case protocol.SourceTool:
		return "tool"
	case protocol.SourceFile:
		return "artifact"
	case protocol.SourceSession:
		return "workflow"
	default:
		return "work_package"
	}
}

// observationID derives a stable, unique contract id for the observation from
// its session and sequence; the journal assigns a unique sequence per session.
func observationID(ev protocol.Event) string {
	return identifier(ev.SessionID) + ":" + strconv.Itoa(ev.Sequence)
}

// occurredAt returns the observation timestamp or a deterministic fallback, so
// the mapped event always satisfies the contracts timestamp requirement.
func occurredAt(ev protocol.Event) string {
	if ev.OccurredAt != "" {
		return ev.OccurredAt
	}
	return zeroTimestamp
}

// actor maps the observation actor onto the contracts actor. An observation
// without an actor is attributed to the observation source itself.
func actor(ev protocol.Event) visualizer.Actor {
	if ev.Actor == "" {
		return visualizer.Actor{Kind: "system", ID: "obsv"}
	}
	return visualizer.Actor{Kind: "agent", ID: identifier(ev.Actor)}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// identifier normalizes s into a contracts identifier: it keeps the permitted
// characters, replaces the rest, and guarantees a leading alphanumeric rune.
func identifier(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isIdentifierRune(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	if !isLeadingRune(rune(out[0])) {
		out = "x" + out
	}
	return out
}

func isIdentifierRune(r rune) bool {
	return isLeadingRune(r) || r == '_' || r == '.' || r == ':' || r == '@' || r == '+' || r == '-'
}

func isLeadingRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// ContractVersion returns the contract version the adapter emits.
func ContractVersion() string { return v1.Version }
