// Package transport is aa-obsv's local observation transport.
//
// A host owns one journal per session. A caller that greets an existing session
// attaches to the current owner instead of starting a second journal, so there
// is exactly one observation record per session. The methods mirror the wire
// protocol: hello, append, replay, subscribe.
package transport

import (
	"fmt"
	"sync"

	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/protocol"
)

// Hello is the response to a session greeting.
type Hello struct {
	Protocol  string `json:"protocol"`
	SessionID string `json:"sessionId"`
	Owner     string `json:"owner"`
	Attached  bool   `json:"attached"`
}

// Ack confirms an append.
type Ack struct {
	Sequence int `json:"sequence"`
}

// Server is the observation transport surface.
type Server interface {
	Hello(sessionID string) (Hello, error)
	Append(sessionID string, ev protocol.Event) (Ack, error)
	Replay(sessionID string, from int) ([]protocol.Event, error)
	Subscribe(sessionID string, from int) (*journal.Subscription, error)
}

// NewJournal opens the journal for a newly owned session.
type NewJournal func(sessionID string) (journal.Journal, error)

type session struct {
	hello Hello
	j     journal.Journal
}

// Local is an in-process transport implementation.
type Local struct {
	ownerID    string
	newJournal NewJournal

	mu       sync.Mutex
	sessions map[string]*session
	closed   bool
}

// NewLocal returns an in-process transport owned by ownerID. When newJournal is
// nil an in-memory journal is used per session.
func NewLocal(ownerID string, newJournal NewJournal) *Local {
	if newJournal == nil {
		newJournal = func(string) (journal.Journal, error) { return journal.NewMemory(), nil }
	}
	return &Local{ownerID: ownerID, newJournal: newJournal, sessions: make(map[string]*session)}
}

// Hello attaches to an existing session or owns a new one.
func (l *Local) Hello(sessionID string) (Hello, error) {
	if sessionID == "" {
		return Hello{}, fmt.Errorf("obsv/transport: sessionId is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return Hello{}, fmt.Errorf("obsv/transport: transport is closed")
	}
	if s, ok := l.sessions[sessionID]; ok {
		hello := s.hello
		hello.Attached = true
		return hello, nil
	}
	j, err := l.newJournal(sessionID)
	if err != nil {
		return Hello{}, err
	}
	hello := Hello{Protocol: protocol.Version, SessionID: sessionID, Owner: l.ownerID, Attached: false}
	l.sessions[sessionID] = &session{hello: hello, j: j}
	return hello, nil
}

// Append stores an event for a session.
func (l *Local) Append(sessionID string, ev protocol.Event) (Ack, error) {
	j, err := l.journal(sessionID)
	if err != nil {
		return Ack{}, err
	}
	stored, err := j.Append(ev)
	if err != nil {
		return Ack{}, err
	}
	return Ack{Sequence: stored.Sequence}, nil
}

// Replay returns events at or after from.
func (l *Local) Replay(sessionID string, from int) ([]protocol.Event, error) {
	j, err := l.journal(sessionID)
	if err != nil {
		return nil, err
	}
	return j.Replay(from)
}

// Subscribe returns a live subscription for a session.
func (l *Local) Subscribe(sessionID string, from int) (*journal.Subscription, error) {
	j, err := l.journal(sessionID)
	if err != nil {
		return nil, err
	}
	return j.Subscribe(from)
}

// Close closes every owned session journal.
func (l *Local) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	sessions := make([]*session, 0, len(l.sessions))
	for _, s := range l.sessions {
		sessions = append(sessions, s)
	}
	l.sessions = make(map[string]*session)
	l.mu.Unlock()

	var firstErr error
	for _, s := range sessions {
		if err := s.j.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (l *Local) journal(sessionID string) (journal.Journal, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("obsv/transport: sessionId is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	s, ok := l.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("obsv/transport: unknown session %q", sessionID)
	}
	return s.j, nil
}

// Client is a session-bound view of a Server.
type Client struct {
	server    Server
	sessionID string
	hello     Hello
}

// Dial greets a session and returns a client bound to it.
func Dial(server Server, sessionID string) (*Client, error) {
	hello, err := server.Hello(sessionID)
	if err != nil {
		return nil, err
	}
	return &Client{server: server, sessionID: sessionID, hello: hello}, nil
}

// Hello returns the greeting recorded at dial time.
func (c *Client) Hello() Hello { return c.hello }

// Attached reports whether the client joined an existing session owner.
func (c *Client) Attached() bool { return c.hello.Attached }

// SessionID returns the bound session.
func (c *Client) SessionID() string { return c.sessionID }

// Append sends an event.
func (c *Client) Append(ev protocol.Event) (Ack, error) {
	return c.server.Append(c.sessionID, ev)
}

// Replay returns events at or after from.
func (c *Client) Replay(from int) ([]protocol.Event, error) {
	return c.server.Replay(c.sessionID, from)
}

// Subscribe returns a live subscription.
func (c *Client) Subscribe(from int) (*journal.Subscription, error) {
	return c.server.Subscribe(c.sessionID, from)
}
