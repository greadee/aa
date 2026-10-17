// Package obsplane composes aa-obsv for the control-plane daemon.
//
// One owner hosts one journal per session. Events are sanitized by the
// protocol allowlist before journaling, delivery is failure-open, and replay,
// subscription, reports, and the Codex translator all ride the composed
// obsv transport. Observation never carries canonical history and never
// changes agent behavior.
package obsplane

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/greadee/aa/obsv"
	"github.com/greadee/aa/obsv/codex"
	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/obsv/report"
	"github.com/greadee/aa/obsv/transport"
)

// Config configures the observation plane.
type Config struct {
	// Owner identifies the process that owns the journals.
	Owner string
	// Dir, when set, stores one durable JSONL journal per session beneath it.
	Dir string
	// Journals overrides journal construction (tests, alternate storage).
	Journals transport.NewJournal
	// Retention bounds the projected view used by reports.
	Retention journal.Policy
}

// Ingested reports the outcome of a Codex translation.
type Ingested struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"`
}

// Plane is the composed observation service.
type Plane struct {
	manager   *obsv.Manager
	retention journal.Policy
}

// New composes an observation plane. It returns an error only when the durable
// journal directory cannot be created; per-session journal failures stay
// failure-open at emit time.
func New(cfg Config) (*Plane, error) {
	sessions := ""
	if cfg.Dir != "" {
		sessions = filepath.Join(cfg.Dir, "sessions")
		if err := os.MkdirAll(sessions, 0o700); err != nil {
			return nil, fmt.Errorf("obsplane: create journal directory: %w", err)
		}
	}
	if cfg.Journals == nil && sessions != "" {
		cfg.Journals = fileJournals(sessions)
	}
	return &Plane{
		manager:   obsv.NewManager(cfg.Owner, cfg.Journals),
		retention: cfg.Retention,
	}, nil
}

// Close closes the transport and every owned session journal.
func (p *Plane) Close() error { return p.manager.Close() }

// Emit sends one observation. It is failure-open: a missing session, an
// invalid event, a full journal, or a panicking observer returns false and
// never blocks or alters the caller.
func (p *Plane) Emit(sessionID string, ev protocol.Event) bool {
	if sessionID == "" {
		return false
	}
	if ev.Version == "" {
		ev.Version = protocol.Version
	}
	if ev.SessionID == "" {
		ev.SessionID = sessionID
	}
	runtime, err := p.manager.Ensure(sessionID)
	if err != nil {
		return false
	}
	return runtime.Send(ev)
}

// Replay returns journaled events at or after from.
func (p *Plane) Replay(sessionID string, from int) ([]protocol.Event, error) {
	runtime, err := p.manager.Ensure(sessionID)
	if err != nil {
		return nil, err
	}
	return runtime.Replay(from)
}

// Subscribe returns a live subscription at or after from.
func (p *Plane) Subscribe(sessionID string, from int) (*journal.Subscription, error) {
	runtime, err := p.manager.Ensure(sessionID)
	if err != nil {
		return nil, err
	}
	return runtime.Subscribe(from)
}

// Report derives a deterministic report over the retained projected view.
func (p *Plane) Report(sessionID string) (report.Report, error) {
	events, err := p.Replay(sessionID, 0)
	if err != nil {
		return report.Report{}, err
	}
	return report.Build(sessionID, applyPolicy(events, p.retention)), nil
}

// IngestCodex translates Codex exec JSONL into observation events and journals
// them. Content-bearing keys are dropped by the protocol sanitizer.
func (p *Plane) IngestCodex(sessionID string, r io.Reader) (Ingested, error) {
	events, err := codex.Translate(sessionID, r)
	if err != nil {
		return Ingested{}, err
	}
	out := Ingested{}
	for _, ev := range events {
		if p.Emit(sessionID, ev) {
			out.Accepted++
		} else {
			out.Dropped++
		}
	}
	return out, nil
}

func applyPolicy(events []protocol.Event, p journal.Policy) []protocol.Event {
	kept := make([]protocol.Event, 0, len(events))
	for _, ev := range events {
		if p.MinSequence > 0 && ev.Sequence < p.MinSequence {
			continue
		}
		kept = append(kept, ev)
	}
	if p.MaxEvents > 0 && len(kept) > p.MaxEvents {
		kept = kept[len(kept)-p.MaxEvents:]
	}
	return kept
}

func fileJournals(dir string) transport.NewJournal {
	return func(sessionID string) (journal.Journal, error) {
		sum := sha256.Sum256([]byte(sessionID))
		name := hex.EncodeToString(sum[:]) + ".jsonl"
		return journal.OpenFile(filepath.Join(dir, name))
	}
}
