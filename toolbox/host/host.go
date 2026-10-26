// Package host is the provider-agnostic tool invocation host.
//
// It looks up a validated manifest, evaluates the caller's execution contract
// through the policy engine, selects the provider for the tool's kind, and runs
// the tool. Invocation is idempotent: repeating a key returns the original
// result. It never grants a capability and never bypasses a denial.
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

// Host is the provider-agnostic invocation host.
type Host struct {
	registry *registry.Registry
	policy   *policy.Engine
	metrics  toolbox.Metrics

	mu    sync.Mutex
	cache map[string]hostCacheEntry
}

type hostCacheEntry struct {
	result    toolbox.Result
	inputHash string
}

// New returns a host over a registry and policy engine. A nil metrics value
// uses a no-op collector so instrumentation never changes behavior.
func New(reg *registry.Registry, eng *policy.Engine, metrics ...toolbox.Metrics) *Host {
	m := toolbox.Metrics(toolbox.NoopMetrics{})
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	}
	return &Host{registry: reg, policy: eng, metrics: m, cache: map[string]hostCacheEntry{}}
}

// Invoke runs a tool under an execution contract. It returns a typed error
// wrapping ErrInvalid, ErrNotFound, ErrDenied, ErrSandbox, or ErrNoProvider.
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

	contractID := contract.ID
	if contractID == "" {
		contractID = inv.ExecutionContractID
	}
	key := toolbox.InvocationKey(inv.ToolID, contractID, inv.IdempotencyKey)
	inputHash := inputFingerprint(inv.Input)

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
	h.mu.Unlock()
	h.metrics.RecordCacheMiss(key)

	output, err := provider.Invoke(ctx, manifest, inv.Input)
	if err != nil {
		return toolbox.Result{}, err
	}
	result := toolbox.Result{ToolID: inv.ToolID, Output: output}

	h.mu.Lock()
	h.cache[key] = hostCacheEntry{result: result, inputHash: inputHash}
	h.mu.Unlock()
	success = true
	return result, nil
}

func inputFingerprint(input map[string]any) string {
	data, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return toolbox.Hash(data)
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
