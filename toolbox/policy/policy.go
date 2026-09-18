// Package policy decides whether a caller may invoke a tool.
//
// A grant is the intersection of the tool manifest's declared capabilities and
// the caller's execution contract, minus any capability the contract explicitly
// denies. Capabilities are never granted implicitly or by default. Required
// sandboxing and network access fail closed. Every decision is appended to an
// ordered audit log.
package policy

import (
	"sort"
	"sync"
	"time"

	toolbox "github.com/greadee/aa/toolbox"
)

// SandboxEnforcer reports whether an isolation kind can actually be enforced on
// this host. A nil enforcer can enforce nothing, so required sandboxing fails
// closed.
type SandboxEnforcer interface {
	CanEnforce(kind string) bool
}

// SandboxDecision records the isolation outcome for one evaluation.
type SandboxDecision struct {
	Required bool   `json:"required"`
	Kind     string `json:"kind,omitempty"`
	Network  bool   `json:"network,omitempty"`
	Allowed  bool   `json:"allowed"`
	Reason   string `json:"reason,omitempty"`
}

// Decision is the outcome of a policy evaluation.
type Decision struct {
	Allowed     bool                 `json:"allowed"`
	ToolID      toolbox.ToolID       `json:"toolId"`
	Granted     []toolbox.Capability `json:"granted"`
	Missing     []toolbox.Capability `json:"missing,omitempty"`
	Denied      []toolbox.Capability `json:"denied,omitempty"`
	Permissions []string             `json:"permissions,omitempty"`
	Sandbox     SandboxDecision      `json:"sandbox"`
	Reason      string               `json:"reason,omitempty"`
}

// AuditEntry records one decision in the ordered audit log.
type AuditEntry struct {
	Seq        int                  `json:"seq"`
	ToolID     toolbox.ToolID       `json:"toolId"`
	ContractID string               `json:"executionContractId,omitempty"`
	Allowed    bool                 `json:"allowed"`
	Granted    []toolbox.Capability `json:"granted,omitempty"`
	Missing    []toolbox.Capability `json:"missing,omitempty"`
	Denied     []toolbox.Capability `json:"denied,omitempty"`
	Sandbox    string               `json:"sandbox,omitempty"`
	Reason     string               `json:"reason,omitempty"`
	At         time.Time            `json:"at"`
}

// Engine evaluates manifest/contract intersections and records an audit log.
type Engine struct {
	mu       sync.Mutex
	enforcer SandboxEnforcer
	clock    toolbox.Clock
	seq      int
	log      []AuditEntry
}

// New returns an engine. A nil enforcer enforces no sandbox; a nil clock uses
// wall time.
func New(enforcer SandboxEnforcer, clock toolbox.Clock) *Engine {
	if clock == nil {
		clock = toolbox.SystemClock{}
	}
	return &Engine{enforcer: enforcer, clock: clock}
}

// Evaluate intersects a manifest's capabilities with an execution contract and
// decides the sandbox, appending the result to the audit log.
func (e *Engine) Evaluate(m toolbox.Manifest, c toolbox.ExecutionContract) Decision {
	grantedSet := map[toolbox.Capability]bool{}
	for _, cap := range c.Capabilities {
		grantedSet[cap] = true
	}
	deniedSet := map[toolbox.Capability]bool{}
	for _, cap := range c.Denied {
		deniedSet[cap] = true
	}

	var granted, missing, denied []toolbox.Capability
	for _, raw := range m.Capabilities {
		cap := toolbox.Capability(raw)
		switch {
		case deniedSet[cap]:
			denied = append(denied, cap)
		case grantedSet[cap]:
			granted = append(granted, cap)
		default:
			missing = append(missing, cap)
		}
	}
	sortCaps(granted)
	sortCaps(missing)
	sortCaps(denied)

	sandbox := e.evaluateSandbox(m, grantedSet)

	decision := Decision{
		ToolID:      toolbox.ToolID(m.ID),
		Granted:     granted,
		Missing:     missing,
		Denied:      denied,
		Permissions: append([]string(nil), m.Permissions...),
		Sandbox:     sandbox,
		Allowed:     len(missing) == 0 && len(denied) == 0 && sandbox.Allowed,
	}
	decision.Reason = reason(decision)
	e.record(decision, c.ID)
	return decision
}

func (e *Engine) evaluateSandbox(m toolbox.Manifest, granted map[toolbox.Capability]bool) SandboxDecision {
	spec := m.Sandbox
	if spec == nil {
		return SandboxDecision{Allowed: true}
	}
	kind := spec.Kind
	if kind == "" {
		kind = "none"
	}
	d := SandboxDecision{Required: spec.Required, Kind: kind, Network: spec.Network, Allowed: true}
	if spec.Network && !granted[toolbox.Capability(toolbox.CapNetworkAccess)] {
		d.Allowed = false
		d.Reason = "network access requires the network_access capability"
		return d
	}
	if spec.Required {
		if kind == "none" {
			d.Allowed = false
			d.Reason = "sandbox is required but no isolation kind was declared"
			return d
		}
		if e.enforcer == nil || !e.enforcer.CanEnforce(kind) {
			d.Allowed = false
			d.Reason = "sandbox kind " + kind + " cannot be enforced"
			return d
		}
	}
	return d
}

func reason(d Decision) string {
	switch {
	case len(d.Denied) > 0:
		return "explicitly denied capability"
	case len(d.Missing) > 0:
		return "caller lacks a declared capability"
	case !d.Sandbox.Allowed:
		return d.Sandbox.Reason
	default:
		return ""
	}
}

func (e *Engine) record(d Decision, contractID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seq++
	entry := AuditEntry{
		Seq:        e.seq,
		ToolID:     d.ToolID,
		ContractID: contractID,
		Allowed:    d.Allowed,
		Granted:    d.Granted,
		Missing:    d.Missing,
		Denied:     d.Denied,
		Sandbox:    d.Sandbox.Kind,
		Reason:     d.Reason,
		At:         e.clock.Now(),
	}
	e.log = append(e.log, entry)
}

// Audit returns a copy of the ordered audit log.
func (e *Engine) Audit() []AuditEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]AuditEntry, len(e.log))
	copy(out, e.log)
	return out
}

func sortCaps(caps []toolbox.Capability) {
	sort.Slice(caps, func(i, j int) bool { return caps[i] < caps[j] })
}
