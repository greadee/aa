package obsplane

import (
	"strings"
	"testing"

	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/protocol"
)

func event(action string) protocol.Event {
	return protocol.Event{
		SourceType: protocol.SourceTool,
		Confidence: protocol.ConfidenceExact,
		Action:     action,
	}
}

func TestEmitReplayAndReport(t *testing.T) {
	p, err := New(Config{Owner: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if !p.Emit("session-1", event("tool.run")) {
		t.Fatal("emit was rejected")
	}
	events, err := p.Replay("session-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sequence != 0 || events[0].SessionID != "session-1" {
		t.Fatalf("events = %+v", events)
	}
	rep, err := p.Report("session-1")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Events != 1 || rep.LastSequence != 0 {
		t.Fatalf("report = %+v", rep)
	}
}

func TestMetadataAllowlistBeforeJournaling(t *testing.T) {
	p, err := New(Config{Owner: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	ev := event("tool.run")
	ev.Metadata = map[string]string{
		"tool":    "bash",
		"content": "secret code",
		"prompt":  "do the thing",
		"token":   "abc",
	}
	if !p.Emit("session-1", ev) {
		t.Fatal("emit was rejected")
	}
	events, err := p.Replay("session-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	md := events[0].Metadata
	if md["tool"] != "bash" {
		t.Fatalf("allowlisted metadata lost: %+v", md)
	}
	for _, denied := range []string{"content", "prompt", "token"} {
		if _, ok := md[denied]; ok {
			t.Fatalf("denied key %q reached the journal: %+v", denied, md)
		}
	}
}

func TestPoisonedObserverIsFailureOpen(t *testing.T) {
	p, err := New(Config{
		Owner: "test",
		Journals: func(string) (journal.Journal, error) {
			return poisonJournal{journal.NewMemory()}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if p.Emit("session-1", event("first")) {
		t.Fatal("poisoned observer accepted an event")
	}
	if p.Emit("session-1", event("second")) {
		t.Fatal("poisoned observer accepted a second event")
	}
}

func TestDurableJournalReopens(t *testing.T) {
	dir := t.TempDir()
	p, err := New(Config{Owner: "test", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Emit("session-1", event("persisted")) {
		t.Fatal("emit was rejected")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(Config{Owner: "test", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	events, err := reopened.Replay("session-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "persisted" {
		t.Fatalf("reopened events = %+v", events)
	}
}

func TestReportAppliesRetention(t *testing.T) {
	p, err := New(Config{Owner: "test", Retention: journal.Policy{MaxEvents: 2}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for i := 0; i < 5; i++ {
		if !p.Emit("session-1", event("step")) {
			t.Fatal("emit was rejected")
		}
	}
	rep, err := p.Report("session-1")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Events != 2 || rep.FirstSequence != 3 || rep.LastSequence != 4 {
		t.Fatalf("report = %+v", rep)
	}
}

func TestIngestCodexDropsContent(t *testing.T) {
	p, err := New(Config{Owner: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	ingested, err := p.IngestCodex("session-1", strings.NewReader(`{"type":"tool_call","tool":"bash","content":"secret"}`+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if ingested.Accepted != 1 || ingested.Dropped != 0 {
		t.Fatalf("ingested = %+v", ingested)
	}
	events, err := p.Replay("session-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "tool_call" {
		t.Fatalf("events = %+v", events)
	}
	if _, ok := events[0].Metadata["content"]; ok {
		t.Fatalf("content reached the journal: %+v", events[0].Metadata)
	}
	if events[0].Metadata["tool"] != "bash" {
		t.Fatalf("metadata = %+v", events[0].Metadata)
	}
}

type poisonJournal struct {
	*journal.Memory
}

func (poisonJournal) Append(protocol.Event) (protocol.Event, error) {
	panic("poisoned observer")
}
