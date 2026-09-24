package browse

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/query"
	"github.com/greadee/aa/memory/store"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/replay"
)

func ev(id string, seq int, project string) visualizer.Event {
	e := visualizer.Event{}
	e.ContractVersion = "1.0"
	e.Kind = "event"
	e.ID = id
	e.ProjectID = project
	e.Sequence = seq
	e.OccurredAt = "2026-01-01T00:00:00Z"
	e.Type = "WORK_PACKAGE_CREATED"
	e.Aggregate = visualizer.EventAggregate{Kind: "work_package", ID: "wp_" + id}
	e.Actor = visualizer.Actor{Kind: "agent", ID: "worker_1"}
	return e
}

func fixture() []visualizer.Event {
	e1 := ev("e1", 1, "p1")
	e2 := ev("e2", 2, "p1")
	e3 := ev("e3", 3, "p2")
	e4 := ev("e4", 4, "p1")
	e4.Payload = map[string]any{"sessionId": "sess_x"}
	return []visualizer.Event{e1, e2, e3, e4}
}

// newBrowser appends events to a real on-disk memory store and returns a
// browser over the memory query seam. Tests exercise the real seam rather than
// a fake event source.
func newBrowser(t *testing.T, events ...visualizer.Event) *Browser {
	t.Helper()
	b, _ := newBrowserWithStore(t, events)
	return b
}

func newBrowserWithStore(t *testing.T, events []visualizer.Event) (*Browser, *store.Store) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	for i, e := range events {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal event %d: %v", i, err)
		}
		if _, err := s.AppendEvent(data); err != nil {
			t.Fatalf("AppendEvent %d: %v", i, err)
		}
	}
	b, err := New(query.New(projection.NewMem(), s))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b, s
}

func TestListGroupsAndSortsSessions(t *testing.T) {
	b := newBrowser(t, fixture()...)
	sessions, err := b.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var ids []visualizer.SessionID
	for _, s := range sessions {
		ids = append(ids, s.ID)
	}
	if !reflect.DeepEqual(ids, []visualizer.SessionID{"p1", "p2", "sess_x"}) {
		t.Fatalf("session ids = %v", ids)
	}
	for _, s := range sessions {
		switch s.ID {
		case "p1":
			if s.EventCount != 2 || s.StartSequence != 1 || s.EndSequence != 2 {
				t.Fatalf("p1 summary = %+v", s)
			}
			if s.ProjectID != "p1" {
				t.Fatalf("p1 project = %q", s.ProjectID)
			}
		case "p2":
			if s.EventCount != 1 {
				t.Fatalf("p2 summary = %+v", s)
			}
		case "sess_x":
			if s.EventCount != 1 {
				t.Fatalf("sess_x summary = %+v", s)
			}
		}
	}
}

func TestLoadReturnsSequenceOrder(t *testing.T) {
	events := fixture()
	// Appended out of order; the canonical log keeps insertion order, so Load
	// must order by event Sequence.
	b := newBrowser(t, events[3], events[1], events[0])
	got, err := b.Load("p1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var seqs []int
	for _, e := range got {
		seqs = append(seqs, e.Sequence)
	}
	if !reflect.DeepEqual(seqs, []int{1, 2}) {
		t.Fatalf("sequences = %v", seqs)
	}
}

func TestLoadUnknownAndInvalid(t *testing.T) {
	b := newBrowser(t, ev("e1", 1, "p1"))
	if _, err := b.Load("ghost"); !errors.Is(err, visualizer.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if _, err := b.Load(""); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestTimelineMatchesReplay(t *testing.T) {
	events := fixture()
	b := newBrowser(t, events...)
	got, err := b.Timeline("p1")
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	want, err := replay.Build("p1", []visualizer.Event{events[0], events[1]})
	if err != nil {
		t.Fatalf("replay.Build: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("timeline differs from replay:\n%+v\n%+v", got, want)
	}
}

func TestSessionKeyPrecedence(t *testing.T) {
	withSession := ev("e1", 1, "p1")
	withSession.Payload = map[string]any{"sessionId": "sess_x"}
	if got := SessionKey(withSession); got != "sess_x" {
		t.Fatalf("SessionKey = %q, want sess_x", got)
	}
	if got := SessionKey(ev("e2", 2, "p1")); got != "p1" {
		t.Fatalf("SessionKey = %q, want p1", got)
	}
	if got := SessionKey(ev("e3", 3, "")); got != DefaultSessionID {
		t.Fatalf("SessionKey = %q, want %q", got, DefaultSessionID)
	}
}

func TestNewRejectsNil(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestListRejectsBadEvent(t *testing.T) {
	// A well-formed log entry that is not a valid visualizer event: the store
	// probe (id/sequence) accepts it, the visualizer decode must reject it.
	b, s := newBrowserWithStore(t, nil)
	line := []byte(`{"id":"e1","sequence":1,"aggregate":"not-an-object"}` + "\n")
	if err := os.WriteFile(s.Layout().EventsPath(), line, 0o600); err != nil {
		t.Fatalf("write corrupt event: %v", err)
	}
	if _, err := b.List(); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestBrowserOverMemoryQuery(t *testing.T) {
	b := newBrowser(t, fixture()...)
	sessions, err := b.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(sessions))
	}
	events, err := b.Load("p1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("p1 events = %d, want 2", len(events))
	}
}
