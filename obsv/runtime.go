package obsv

import (
	"fmt"
	"sync"

	"github.com/greadee/aa/obsv/emit"
	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/obsv/report"
	"github.com/greadee/aa/obsv/transport"
)

// Runtime is a session-bound observation handle.
type Runtime struct {
	client  *transport.Client
	emitter *emit.Emitter
}

// SessionID returns the bound session.
func (r *Runtime) SessionID() string { return r.client.SessionID() }

// Hello returns the session greeting.
func (r *Runtime) Hello() transport.Hello { return r.client.Hello() }

// Attached reports whether this handle joined an existing session owner.
func (r *Runtime) Attached() bool { return r.client.Attached() }

// Send delivers an observation event. It is sanitized and validated by the
// protocol, then written failure-open: a rejected event returns false and never
// blocks or panics the caller.
func (r *Runtime) Send(ev protocol.Event) bool { return r.emitter.Emit(ev) }

// Replay returns journaled events at or after from.
func (r *Runtime) Replay(from int) ([]protocol.Event, error) { return r.client.Replay(from) }

// Subscribe returns a live subscription at or after from.
func (r *Runtime) Subscribe(from int) (*journal.Subscription, error) {
	return r.client.Subscribe(from)
}

// Report returns a deterministic work report for the session.
func (r *Runtime) Report() (report.Report, error) {
	events, err := r.client.Replay(0)
	if err != nil {
		return report.Report{}, err
	}
	return report.Build(r.client.SessionID(), events), nil
}

// Stats returns the emitter counters.
func (r *Runtime) Stats() emit.Stats { return r.emitter.Stats() }

// clientTarget adapts a transport client to an emit.Target.
type clientTarget struct{ client *transport.Client }

func (t clientTarget) Append(ev protocol.Event) error {
	_, err := t.client.Append(ev)
	return err
}

// Manager owns one local transport and returns one runtime per session, so a
// repeated Ensure attaches to the existing session instead of opening a second
// journal.
type Manager struct {
	owner string
	host  *transport.Local

	mu       sync.Mutex
	runtimes map[string]*Runtime
	closed   bool
}

// NewManager returns a manager for a session owner. When newJournal is nil an
// in-memory journal is used per session; pass a file-backed constructor for a
// durable journal.
func NewManager(owner string, newJournal transport.NewJournal) *Manager {
	if owner == "" {
		owner = "kernel"
	}
	return &Manager{
		owner:    owner,
		host:     transport.NewLocal(owner, newJournal),
		runtimes: make(map[string]*Runtime),
	}
}

// Ensure returns the runtime for a session, creating and owning it on first use
// and attaching to it thereafter.
func (m *Manager) Ensure(sessionID string) (*Runtime, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("obsv: sessionId is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, fmt.Errorf("obsv: manager is closed")
	}
	if r, ok := m.runtimes[sessionID]; ok {
		return r, nil
	}
	client, err := transport.Dial(m.host, sessionID)
	if err != nil {
		return nil, err
	}
	r := &Runtime{client: client, emitter: emit.New(clientTarget{client: client})}
	m.runtimes[sessionID] = r
	return r, nil
}

// Close closes the transport and every owned session journal.
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.runtimes = make(map[string]*Runtime)
	m.mu.Unlock()
	return m.host.Close()
}

var defaultManager = NewManager("kernel", nil)

// Ensure returns the runtime for a session from the process-wide manager.
func Ensure(sessionID string) (*Runtime, error) { return defaultManager.Ensure(sessionID) }

// Send delivers an event to a session from the process-wide manager.
func Send(sessionID string, ev protocol.Event) (bool, error) {
	r, err := defaultManager.Ensure(sessionID)
	if err != nil {
		return false, err
	}
	return r.Send(ev), nil
}

// Replay returns events from a session in the process-wide manager.
func Replay(sessionID string, from int) ([]protocol.Event, error) {
	r, err := defaultManager.Ensure(sessionID)
	if err != nil {
		return nil, err
	}
	return r.Replay(from)
}

// Subscribe returns a live subscription from the process-wide manager.
func Subscribe(sessionID string, from int) (*journal.Subscription, error) {
	r, err := defaultManager.Ensure(sessionID)
	if err != nil {
		return nil, err
	}
	return r.Subscribe(from)
}
