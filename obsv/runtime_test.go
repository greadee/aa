package obsv

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/protocol"
)

func event() protocol.Event {
	return protocol.Event{
		Version:    protocol.Version,
		SessionID:  "ses_1",
		SourceType: protocol.SourceTool,
		Confidence: protocol.ConfidenceObserved,
		Metadata:   map[string]string{"tool": "rg", "prompt": "leak"},
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	m := NewManager("kernel", nil)
	defer m.Close()
	first, err := m.Ensure("ses_1")
	if err != nil {
		t.Fatalf("Ensure() = %v", err)
	}
	second, err := m.Ensure("ses_1")
	if err != nil {
		t.Fatalf("Ensure() = %v", err)
	}
	if first != second {
		t.Fatal("Ensure returned two runtimes for one session")
	}
}

func TestSendReplayReport(t *testing.T) {
	m := NewManager("kernel", nil)
	defer m.Close()
	r, err := m.Ensure("ses_1")
	if err != nil {
		t.Fatalf("Ensure() = %v", err)
	}
	if !r.Send(event()) {
		t.Fatal("Send() = false, want true")
	}

	events, err := r.Replay(0)
	if err != nil {
		t.Fatalf("Replay() = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Replay() len = %d, want 1", len(events))
	}
	if _, ok := events[0].Metadata["prompt"]; ok {
		t.Fatal("prompt metadata survived Send")
	}
	if events[0].Metadata["tool"] != "rg" {
		t.Fatal("allowlisted metadata lost")
	}

	rep, err := r.Report()
	if err != nil {
		t.Fatalf("Report() = %v", err)
	}
	if rep.Events != 1 || rep.SessionID != "ses_1" {
		t.Fatalf("Report() = %+v", rep)
	}
}

func TestSendRejectsInvalidFailureOpen(t *testing.T) {
	m := NewManager("kernel", nil)
	defer m.Close()
	r, _ := m.Ensure("ses_1")
	bad := event()
	bad.SourceType = "network"
	if r.Send(bad) {
		t.Fatal("Send(invalid) = true, want false")
	}
	events, _ := r.Replay(0)
	if len(events) != 0 {
		t.Fatalf("Replay() len = %d, want 0", len(events))
	}
	if r.Stats().Dropped != 1 {
		t.Fatalf("Stats().Dropped = %d, want 1", r.Stats().Dropped)
	}
}

func TestSubscribeReceivesLive(t *testing.T) {
	m := NewManager("kernel", nil)
	defer m.Close()
	r, _ := m.Ensure("ses_1")
	sub, err := r.Subscribe(0)
	if err != nil {
		t.Fatalf("Subscribe() = %v", err)
	}
	defer sub.Close()
	r.Send(event())
	select {
	case ev := <-sub.Events():
		if ev.Sequence != 0 {
			t.Fatalf("sequence = %d, want 0", ev.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("no subscribed event")
	}
}

func TestDurableJournalSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	newJournal := func(sessionID string) (journal.Journal, error) {
		return journal.OpenFile(filepath.Join(dir, sessionID+".jsonl"))
	}

	m := NewManager("kernel", newJournal)
	r, err := m.Ensure("ses_1")
	if err != nil {
		t.Fatalf("Ensure() = %v", err)
	}
	r.Send(event())
	if err := m.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}

	m2 := NewManager("kernel", newJournal)
	defer m2.Close()
	r2, err := m2.Ensure("ses_1")
	if err != nil {
		t.Fatalf("Ensure() = %v", err)
	}
	events, err := r2.Replay(0)
	if err != nil {
		t.Fatalf("Replay() = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Replay() len = %d, want 1", len(events))
	}
}

func TestManagerClose(t *testing.T) {
	m := NewManager("kernel", nil)
	if err := m.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if _, err := m.Ensure("ses_1"); err == nil {
		t.Fatal("Ensure after Close = nil, want error")
	}
}

func TestPackageFacade(t *testing.T) {
	r, err := Ensure("ses_facade")
	if err != nil {
		t.Fatalf("Ensure() = %v", err)
	}
	if r.SessionID() != "ses_facade" {
		t.Fatalf("SessionID = %q", r.SessionID())
	}
	ok, err := Send("ses_facade", event())
	if err != nil || !ok {
		t.Fatalf("Send() = %v, %v", ok, err)
	}
	events, err := Replay("ses_facade", 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("Replay() = %v, %v", events, err)
	}
	if _, err := Ensure(""); err == nil {
		t.Fatal("Ensure(\"\") = nil, want error")
	}
}
