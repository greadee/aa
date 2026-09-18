package host

import (
	"context"
	"sync"

	toolbox "github.com/greadee/aa/toolbox"
)

// Call records one provider invocation.
type Call struct {
	Seq    int
	ToolID toolbox.ToolID
	Input  map[string]any
}

// FakeProvider is a deterministic in-memory tool provider for tests. By default
// it echoes its input plus the manifest identity; specific results and errors
// can be configured per tool.
type FakeProvider struct {
	kind string

	mu      sync.Mutex
	results map[toolbox.ToolID]map[string]any
	errs    map[toolbox.ToolID]error
	calls   []Call
}

// NewFakeProvider returns a fake provider for a tool kind.
func NewFakeProvider(kind string) *FakeProvider {
	return &FakeProvider{
		kind:    kind,
		results: map[toolbox.ToolID]map[string]any{},
		errs:    map[toolbox.ToolID]error{},
	}
}

// Kind implements registry.Provider.
func (f *FakeProvider) Kind() string { return f.kind }

// SetResult configures the output for a tool.
func (f *FakeProvider) SetResult(id toolbox.ToolID, out map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[id] = out
}

// SetError configures an error for a tool.
func (f *FakeProvider) SetError(id toolbox.ToolID, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[id] = err
}

// Invoke implements registry.Provider with deterministic output.
func (f *FakeProvider) Invoke(_ context.Context, m toolbox.Manifest, input map[string]any) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, Call{Seq: len(f.calls) + 1, ToolID: toolbox.ToolID(m.ID), Input: copyMap(input)})
	if err, ok := f.errs[toolbox.ToolID(m.ID)]; ok {
		return nil, err
	}
	if out, ok := f.results[toolbox.ToolID(m.ID)]; ok {
		return copyMap(out), nil
	}
	echo := copyMap(input)
	if echo == nil {
		echo = map[string]any{}
	}
	echo["tool"] = m.ID
	echo["kind"] = m.ToolKind
	return echo, nil
}

// Calls returns a copy of the recorded calls in order.
func (f *FakeProvider) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Call, len(f.calls))
	copy(out, f.calls)
	return out
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
