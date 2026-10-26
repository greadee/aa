// Package host is the provider-agnostic tool invocation host.
//
// It looks up a validated manifest, evaluates the caller's execution contract
// through the policy engine, selects the provider for the tool's kind, and runs
// the tool. Invocation is idempotent when the caller supplies an explicit key:
// repeating the same request returns the original result. It never grants a
// capability and never bypasses a denial.
package host

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
)

const defaultCacheLimit = 10000

// Host is the provider-agnostic invocation host.
type Host struct {
	registry *registry.Registry
	policy   *policy.Engine
	metrics  toolbox.Metrics

	mu         sync.Mutex
	cache      map[string]hostCacheEntry
	cacheOrder []string
	inflight   map[string]*hostInflight
}

type hostCacheEntry struct {
	result    toolbox.Result
	inputHash string
}

type hostInflight struct {
	done      chan struct{}
	result    toolbox.Result
	err       error
	inputHash string
}

// New returns a host over a registry and policy engine. A nil metrics value
// uses a no-op collector so instrumentation never changes behavior.
func New(reg *registry.Registry, eng *policy.Engine, metrics ...toolbox.Metrics) *Host {
	m := toolbox.Metrics(toolbox.NoopMetrics{})
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	}
	return &Host{
		registry: reg,
		policy:   eng,
		metrics:  m,
		cache:    map[string]hostCacheEntry{},
		inflight: map[string]*hostInflight{},
	}
}

// Invoke runs a tool under an execution contract. It returns a typed error
// wrapping ErrInvalid, ErrNotFound, ErrDenied, ErrSandbox, ErrConflict, or
// ErrNoProvider.
func (h *Host) Invoke(ctx context.Context, inv toolbox.Invocation, contract toolbox.ExecutionContract) (toolbox.Result, error) {
	if inv.ToolID == "" {
		return toolbox.Result{}, fmt.Errorf("%w: toolId is required", toolbox.ErrInvalid)
	}
	start := time.Now()
	manifest, ok := h.registry.Get(inv.ToolID)
	if !ok {
		h.metrics.RecordInvocation(inv.ToolID, "", time.Since(start), false)
		return toolbox.Result{}, fmt.Errorf("%w: tool %q", toolbox.ErrNotFound, inv.ToolID)
	}
	success := false
	defer func() { h.metrics.RecordInvocation(inv.ToolID, manifest.ToolKind, time.Since(start), success) }()

	if h.policy != nil {
		decision := h.policy.Evaluate(manifest, contract)
		if !decision.Allowed {
			if !decision.Sandbox.Allowed {
				return toolbox.Result{}, fmt.Errorf("%w: %s", toolbox.ErrSandbox, decision.Reason)
			}
			return toolbox.Result{}, fmt.Errorf("%w: %s", toolbox.ErrDenied, decision.Reason)
		}
	}
	provider, ok := h.registry.ProviderForManifest(manifest)
	if !ok {
		return toolbox.Result{}, fmt.Errorf("%w: %q", toolbox.ErrNoProvider, manifest.ToolKind)
	}

	// Without an explicit key there is no idempotency contract to enforce.
	// Execute every call instead of silently replaying the first result.
	if inv.IdempotencyKey == "" {
		output, err := invokeProvider(ctx, provider, manifest, inv.Input)
		if err != nil {
			return toolbox.Result{}, err
		}
		success = true
		return toolbox.Result{ToolID: inv.ToolID, Output: output}, nil
	}

	contractID := contract.ID
	if contractID == "" {
		contractID = inv.ExecutionContractID
	}
	key := toolbox.InvocationKey(inv.ToolID, contractID, inv.IdempotencyKey)
	inputHash, err := inputFingerprint(inv.Input)
	if err != nil {
		return toolbox.Result{}, fmt.Errorf("%w: input: %v", toolbox.ErrInvalid, err)
	}

	h.mu.Lock()
	if cached, found := h.cache[key]; found {
		h.mu.Unlock()
		if cached.inputHash != inputHash {
			return toolbox.Result{}, fmt.Errorf("%w: idempotency key reused with a different input", toolbox.ErrConflict)
		}
		h.metrics.RecordCacheHit(key)
		result := cached.result
		result.Replayed = true
		success = true
		return result, nil
	}
	if call, found := h.inflight[key]; found {
		if call.inputHash != inputHash {
			h.mu.Unlock()
			return toolbox.Result{}, fmt.Errorf("%w: idempotency key reused with a different input", toolbox.ErrConflict)
		}
		done := call.done
		h.mu.Unlock()
		select {
		case <-done:
			h.mu.Lock()
			result, err := call.result, call.err
			h.mu.Unlock()
			if err != nil {
				return toolbox.Result{}, err
			}
			result.Replayed = true
			success = true
			return result, nil
		case <-ctx.Done():
			return toolbox.Result{}, ctx.Err()
		}
	}
	call := &hostInflight{done: make(chan struct{}), inputHash: inputHash}
	h.inflight[key] = call
	h.mu.Unlock()
	h.metrics.RecordCacheMiss(key)

	output, err := invokeProvider(ctx, provider, manifest, inv.Input)
	result := toolbox.Result{ToolID: inv.ToolID, Output: output}

	h.mu.Lock()
	call.result, call.err = result, err
	if err == nil {
		h.storeLocked(key, hostCacheEntry{result: result, inputHash: inputHash})
	}
	delete(h.inflight, key)
	close(call.done)
	h.mu.Unlock()

	if err != nil {
		return toolbox.Result{}, err
	}
	success = true
	return result, nil
}

// storeLocked inserts a cache entry and evicts the oldest entry when the cache
// exceeds its bound. Callers must hold h.mu.
func (h *Host) storeLocked(key string, entry hostCacheEntry) {
	if _, exists := h.cache[key]; !exists {
		h.cacheOrder = append(h.cacheOrder, key)
	}
	h.cache[key] = entry
	for len(h.cache) > defaultCacheLimit && len(h.cacheOrder) > 0 {
		oldest := h.cacheOrder[0]
		h.cacheOrder = h.cacheOrder[1:]
		delete(h.cache, oldest)
	}
}

func invokeProvider(ctx context.Context, provider registry.Provider, manifest toolbox.Manifest, input map[string]any) (output map[string]any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: provider panic: %v", toolbox.ErrFailed, recovered)
		}
	}()
	return provider.Invoke(ctx, manifest, input)
}

// inputFingerprint canonicalizes nil and empty input to the same value and
// reports serialization errors instead of silently treating distinct inputs as
// equal.
func inputFingerprint(input map[string]any) (string, error) {
	if input == nil {
		input = map[string]any{}
	}
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return toolbox.Hash(data), nil
}

// Audit returns the ordered policy audit log.
func (h *Host) Audit() []policy.AuditEntry {
	if h.policy == nil {
		return nil
	}
	return h.policy.Audit()
}

// ListTools returns registered manifests filtered by kind (empty means all).
func (h *Host) ListTools(kind string) []toolbox.Manifest {
	return h.registry.List(kind)
}
