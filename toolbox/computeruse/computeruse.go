// Package computeruse provides the computer-use capability as a toolbox tool
// provider.
//
// It declares the authority a computer-use tool needs (screen capture and input
// injection) as manifest permissions and provides a provider that runs actions
// against a Driver. The implementation carries no orchestration policy: it
// validates the action, fails closed when the manifest does not declare the
// permission the action needs, and delegates isolation to the toolbox policy
// engine (which requires an enforceable sandbox before allowing a call).
package computeruse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	toolbox "github.com/greadee/aa/toolbox"
)

// Permissions a computer-use tool declares. They are free-form strings on the
// manifest, so no contract change is needed.
const (
	PermissionScreenCapture  = "screen_capture"
	PermissionInputInjection = "input_injection"
)

// ActionKind enumerates computer-use actions.
type ActionKind string

// Action kinds.
const (
	ActionScreenshot ActionKind = "screenshot"
	ActionClick      ActionKind = "click"
	ActionType       ActionKind = "type"
	ActionKey        ActionKind = "key"
	ActionScroll     ActionKind = "scroll"
)

// Action is one computer-use action.
type Action struct {
	Kind   ActionKind `json:"kind"`
	X      int        `json:"x,omitempty"`
	Y      int        `json:"y,omitempty"`
	Text   string     `json:"text,omitempty"`
	Key    string     `json:"key,omitempty"`
	Amount int        `json:"amount,omitempty"`
}

// Validate checks the action shape.
func (a Action) Validate() error {
	switch a.Kind {
	case ActionScreenshot:
	case ActionClick, ActionScroll:
		if a.X < 0 || a.Y < 0 {
			return fmt.Errorf("coordinates must not be negative")
		}
	case ActionType:
		if a.Text == "" {
			return fmt.Errorf("text is required")
		}
	case ActionKey:
		if a.Key == "" {
			return fmt.Errorf("key is required")
		}
	default:
		return fmt.Errorf("unknown action kind %q", a.Kind)
	}
	return nil
}

// RequiredPermission returns the manifest permission the action needs.
func (a Action) RequiredPermission() string {
	if a.Kind == ActionScreenshot {
		return PermissionScreenCapture
	}
	return PermissionInputInjection
}

// Artifact is an output produced by an action (for example a screenshot).
type Artifact struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Hash  string `json:"hash,omitempty"`
	Bytes []byte `json:"bytes,omitempty"`
}

// Outcome is the result of one action.
type Outcome struct {
	Output    map[string]any `json:"output,omitempty"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
}

// Driver performs computer-use actions. Implementations own the OS interaction;
// this package owns the capability/permission contract, not the mechanics.
type Driver interface {
	Perform(ctx context.Context, action Action) (Outcome, error)
	Close() error
}

// Fake is a deterministic Driver for tests and pilots.
type Fake struct {
	mu      sync.Mutex
	results map[ActionKind]Outcome
	errs    map[ActionKind]error
	calls   []Action
	closed  bool
}

// NewFake returns an empty fake driver.
func NewFake() *Fake {
	return &Fake{results: map[ActionKind]Outcome{}, errs: map[ActionKind]error{}}
}

// Script configures the outcome for an action kind.
func (f *Fake) Script(kind ActionKind, out Outcome) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[kind] = out
}

// ScriptError configures an error for an action kind.
func (f *Fake) ScriptError(kind ActionKind, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[kind] = err
}

// Perform records the action and returns the scripted outcome. It honors
// context cancellation.
func (f *Fake) Perform(ctx context.Context, action Action) (Outcome, error) {
	f.mu.Lock()
	f.calls = append(f.calls, action)
	err, scriptedErr := f.errs[action.Kind]
	out, scriptedOut := f.results[action.Kind]
	f.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return Outcome{}, err
	}
	if scriptedErr {
		return Outcome{}, err
	}
	if scriptedOut {
		return out, nil
	}
	return Outcome{Output: map[string]any{"action": string(action.Kind)}}, nil
}

// Close marks the driver closed.
func (f *Fake) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// Calls returns a copy of the recorded actions in order.
func (f *Fake) Calls() []Action {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Action, len(f.calls))
	copy(out, f.calls)
	return out
}

// Provider implements registry.Provider for builtin computer-use tools.
type Provider struct {
	driver Driver
}

// NewProvider returns a provider over a driver.
func NewProvider(driver Driver) (*Provider, error) {
	if driver == nil {
		return nil, fmt.Errorf("%w: driver is required", toolbox.ErrInvalid)
	}
	return &Provider{driver: driver}, nil
}

// Kind implements registry.Provider.
func (p *Provider) Kind() string { return toolbox.KindBuiltin }

// Invoke validates the action, enforces the declared permission, and runs it.
func (p *Provider) Invoke(ctx context.Context, manifest toolbox.Manifest, input map[string]any) (map[string]any, error) {
	action, err := decodeAction(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", toolbox.ErrInvalid, err)
	}
	permission := action.RequiredPermission()
	if !declaresPermission(manifest, permission) {
		return nil, fmt.Errorf("%w: action %q requires permission %q", toolbox.ErrDenied, action.Kind, permission)
	}
	outcome, err := p.driver.Perform(ctx, action)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", toolbox.ErrFailed, err)
	}
	output := outcome.Output
	if output == nil {
		output = map[string]any{}
	}
	output["permission"] = permission
	if len(outcome.Artifacts) > 0 {
		artifacts := make([]map[string]any, 0, len(outcome.Artifacts))
		for _, artifact := range outcome.Artifacts {
			artifacts = append(artifacts, map[string]any{
				"id": artifact.ID, "kind": artifact.Kind, "hash": artifact.Hash,
			})
		}
		output["artifacts"] = artifacts
	}
	return output, nil
}

// Manifest returns a validated builtin computer-use tool manifest. It requires a
// process sandbox, so the policy engine only allows it when isolation can be
// enforced.
func Manifest(id string) toolbox.Manifest {
	return toolbox.Manifest{
		Envelope:     toolbox.Envelope{ContractVersion: "2.0", Kind: "tool_manifest", ID: id},
		Name:         id,
		ToolKind:     toolbox.KindBuiltin,
		Version:      "1.0",
		Description:  "Computer-use: screen capture and input injection",
		Capabilities: []string{string(toolbox.CapReadProject), string(toolbox.CapCreateArtifact)},
		Permissions:  []string{PermissionScreenCapture, PermissionInputInjection},
		Sandbox:      &toolbox.SandboxSpec{Required: true, Kind: "process"},
	}
}

func decodeAction(input map[string]any) (Action, error) {
	raw, ok := input["action"]
	if !ok {
		return Action{}, fmt.Errorf("action is required")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return Action{}, err
	}
	var action Action
	if err := json.Unmarshal(encoded, &action); err != nil {
		return Action{}, err
	}
	if err := action.Validate(); err != nil {
		return Action{}, err
	}
	return action, nil
}

func declaresPermission(manifest toolbox.Manifest, permission string) bool {
	for _, declared := range manifest.Permissions {
		if declared == permission {
			return true
		}
	}
	return false
}
