// Package command parses console palette commands and records an idempotent
// audit trail of applied commands.
package command

import (
	"errors"
	"strings"
	"sync"
)

// Verbs.
const (
	Open    = "open"
	Refresh = "refresh"
	Submit  = "submit"
	Approve = "approve"
	Create  = "create-project"
	Project = "project"
	Help    = "help"
	Quit    = "quit"
	Exit    = "exit"
)

// ErrEmpty is returned for an empty command line.
var ErrEmpty = errors.New("command: empty")

// Action is a parsed console command.
type Action struct {
	Verb string
	Args []string
	Raw  string
}

// Parse parses a palette line, with or without a leading ':'.
func Parse(line string) (Action, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, ":")
	line = strings.TrimSpace(line)
	if line == "" {
		return Action{}, ErrEmpty
	}
	fields := strings.Fields(line)
	return Action{Verb: strings.ToLower(fields[0]), Args: fields[1:], Raw: line}, nil
}

// Entry is one audited command application.
type Entry struct {
	Key  string
	Verb string
	Args []string
}

// Audit records applied commands and deduplicates by key (idempotent writes).
type Audit struct {
	mu      sync.Mutex
	entries []Entry
	seen    map[string]bool
}

// NewAudit returns an empty audit log.
func NewAudit() *Audit {
	return &Audit{seen: map[string]bool{}}
}

// Apply records a command. It returns false when the key was already applied
// (a no-op retry), true when newly applied.
func (a *Audit) Apply(key string, act Action) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if key != "" {
		if a.seen[key] {
			return false
		}
		a.seen[key] = true
	}
	a.entries = append(a.entries, Entry{Key: key, Verb: act.Verb, Args: append([]string(nil), act.Args...)})
	return true
}

// Entries returns a copy of the audit log.
func (a *Audit) Entries() []Entry {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Entry(nil), a.entries...)
}
