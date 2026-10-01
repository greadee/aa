// Package observability provides structured, correlated events across the
// execution seams and the identity chain that attributes them.
//
// The identity chain is organization → project → subtask → crew → worker →
// {role → team, model}. Events carry a correlation id, and accounting aggregates
// tokens, cost, and compute per chain. Everything is deterministic over the
// recorded events; no model is called.
package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Seam event names.
const (
	TaskReceived        = "task.received"
	ContextNeeds        = "context.needs"
	Retrieval           = "retrieval"
	ContextAssembly     = "context.assembly"
	RoleDecision        = "allocation.role"
	CapabilityDecision  = "allocation.capability"
	ParallelismDecision = "allocation.parallelism"
	ExecutionPlan       = "plan.execution"
	SchedulerDispatch   = "scheduler.dispatch"
	RuntimeStart        = "runtime.start"
	RuntimeStop         = "runtime.stop"
	SandboxEvent        = "sandbox"
	ToolCall            = "tool.call"
	AgentResult         = "agent.result"
	EvaluationLearning  = "evaluation.learning"
	FailureRetry        = "failure.retry"
)

// Identity is the end-to-end attribution of a piece of work.
type Identity struct {
	Organization string `json:"organization"`
	Project      string `json:"project"`
	Subtask      string `json:"subtask"`
	Crew         string `json:"crew,omitempty"`
	Worker       string `json:"worker,omitempty"`
	Role         string `json:"role,omitempty"`
	Team         string `json:"team,omitempty"`
	Model        string `json:"model,omitempty"`
}

// Validate checks that the chain is present.
func (i Identity) Validate() error {
	if i.Organization == "" || i.Project == "" || i.Subtask == "" {
		return fmt.Errorf("observability: organization, project, and subtask are required")
	}
	if i.Worker == "" && i.Role == "" {
		return fmt.Errorf("observability: a worker or role is required")
	}
	return nil
}

// Chain returns the canonical, stable chain string.
func (i Identity) Chain() string {
	return strings.Join([]string{
		i.Organization, i.Project, i.Subtask, i.Crew, i.Worker, i.Role, i.Team, i.Model,
	}, "/")
}

// Event is one structured observation.
type Event struct {
	Name          string         `json:"name"`
	CorrelationID string         `json:"correlationId"`
	Identity      Identity       `json:"identity"`
	Tokens        int            `json:"tokens,omitempty"`
	CostUSD       float64        `json:"costUsd,omitempty"`
	ComputeMS     int            `json:"computeMs,omitempty"`
	Failed        bool           `json:"failed,omitempty"`
	Retry         bool           `json:"retry,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

// Recorder receives structured events.
type Recorder interface {
	Record(event Event) error
}

// MemRecorder is a deterministic in-memory recorder.
type MemRecorder struct {
	mu     sync.Mutex
	events []Event
}

// NewMemRecorder returns an empty recorder.
func NewMemRecorder() *MemRecorder { return &MemRecorder{} }

// Record appends an event, validating its name and identity.
func (r *MemRecorder) Record(event Event) error {
	if event.Name == "" {
		return fmt.Errorf("observability: event name is required")
	}
	if err := event.Identity.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

// Events returns a copy of the recorded events in order.
func (r *MemRecorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}

// Metrics summarizes recorded evidence.
type Metrics struct {
	Events    int     `json:"events"`
	Failed    int     `json:"failed"`
	Retries   int     `json:"retries"`
	Tokens    int     `json:"tokens"`
	CostUSD   float64 `json:"costUsd"`
	ComputeMS int     `json:"computeMs"`
}

// Summarize aggregates metrics over events deterministically.
func Summarize(events []Event) Metrics {
	var m Metrics
	for _, event := range events {
		m.Events++
		if event.Failed {
			m.Failed++
		}
		if event.Retry {
			m.Retries++
		}
		m.Tokens += event.Tokens
		m.CostUSD += event.CostUSD
		m.ComputeMS += event.ComputeMS
	}
	return m
}

// Account is accounting for one identity chain.
type Account struct {
	Chain   string  `json:"chain"`
	Metrics Metrics `json:"metrics"`
}

// Accounting aggregates metrics per identity chain, sorted by chain.
func Accounting(events []Event) []Account {
	byChain := map[string]Metrics{}
	for _, event := range events {
		chain := event.Identity.Chain()
		m := byChain[chain]
		m.Events++
		if event.Failed {
			m.Failed++
		}
		if event.Retry {
			m.Retries++
		}
		m.Tokens += event.Tokens
		m.CostUSD += event.CostUSD
		m.ComputeMS += event.ComputeMS
		byChain[chain] = m
	}
	out := make([]Account, 0, len(byChain))
	for chain, m := range byChain {
		out = append(out, Account{Chain: chain, Metrics: m})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Chain < out[j].Chain })
	return out
}
