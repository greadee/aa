package journal

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/greadee/aa/obsv/protocol"
)

func event() protocol.Event {
	return protocol.Event{
		Version:    protocol.Version,
		SessionID:  "ses_1",
		SourceType: protocol.SourceTool,
		Confidence: protocol.ConfidenceObserved,
	}
}

func TestAppendAssignsOrderedSequence(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	for i := 0; i < 3; i++ {
		stored, err := m.Append(event())
		if err != nil {
			t.Fatalf("Append() = %v", err)
		}
		if stored.Sequence != i {
			t.Fatalf("sequence = %d, want %d", stored.Sequence, i)
		}
	}
	if m.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", m.Len())
	}
}

func TestAppendRejectsInvalid(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	ev := event()
	ev.SourceType = "network"
	if _, err := m.Append(ev); err == nil {
		t.Fatal("Append(invalid) = nil, want error")
	}
	if m.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", m.Len())
	}
}

func TestReplayFromCursor(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	for i := 0; i < 5; i++ {
		if _, err := m.Append(event()); err != nil {
			t.Fatalf("Append() = %v", err)
		}
	}
	got, err := m.Replay(3)
	if err != nil {
		t.Fatalf("Replay() = %v", err)
	}
	if len(got) != 2 || got[0].Sequence != 3 || got[1].Sequence != 4 {
		t.Fatalf("Replay(3) = %v, want sequences 3,4", got)
	}
}

func TestSubscribeDeliversBacklogAndLive(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	if _, err := m.Append(event()); err != nil {
		t.Fatalf("Append() = %v", err)
	}
	sub, err := m.Subscribe(0)
	if err != nil {
		t.Fatalf("Subscribe() = %v", err)
	}
	defer sub.Close()

	select {
	case ev := <-sub.Events():
		if ev.Sequence != 0 {
			t.Fatalf("backlog sequence = %d, want 0", ev.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("no backlog event")
	}

	if _, err := m.Append(event()); err != nil {
		t.Fatalf("Append() = %v", err)
	}
	select {
	case ev := <-sub.Events():
		if ev.Sequence != 1 {
			t.Fatalf("live sequence = %d, want 1", ev.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("no live event")
	}
}

func TestSubscribeDropsWhenLagging(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	sub, err := m.Subscribe(0)
	if err != nil {
		t.Fatalf("Subscribe() = %v", err)
	}
	defer sub.Close()
	// Never read: the channel fills, then delivery must drop rather than block.
	for i := 0; i < 200; i++ {
		if _, err := m.Append(event()); err != nil {
			t.Fatalf("Append() = %v", err)
		}
	}
	if sub.Dropped() == 0 {
		t.Fatal("Dropped() = 0, want > 0")
	}
}

func TestTrimBoundsProjectedView(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	for i := 0; i < 10; i++ {
		if _, err := m.Append(event()); err != nil {
			t.Fatalf("Append() = %v", err)
		}
	}
	got := m.Trim(Policy{MaxEvents: 3, MinSequence: 2})
	if len(got) != 3 {
		t.Fatalf("Trim() len = %d, want 3", len(got))
	}
	if got[0].Sequence != 7 || got[2].Sequence != 9 {
		t.Fatalf("Trim() = %d..%d, want 7..9", got[0].Sequence, got[2].Sequence)
	}
	if m.Len() != 10 {
		t.Fatalf("Trim modified the journal: Len() = %d, want 10", m.Len())
	}
}

func TestFileJournalSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "obsv.jsonl")
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile() = %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := f.Append(event()); err != nil {
			t.Fatalf("Append() = %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}

	reopened, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile() = %v", err)
	}
	defer reopened.Close()
	got, err := reopened.Replay(0)
	if err != nil {
		t.Fatalf("Replay() = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Replay() len = %d, want 3", len(got))
	}
	for i, ev := range got {
		if ev.Sequence != i {
			t.Fatalf("sequence[%d] = %d, want %d", i, ev.Sequence, i)
		}
	}
	// The next append continues the durable sequence.
	stored, err := reopened.Append(event())
	if err != nil {
		t.Fatalf("Append() = %v", err)
	}
	if stored.Sequence != 3 {
		t.Fatalf("sequence = %d, want 3", stored.Sequence)
	}
}
