package toolbox

import (
	"sync"
	"time"
)

// Metrics records toolbox execution and policy outcomes. Implementations must
// be safe for concurrent use. NoopMetrics is the default.
type Metrics interface {
	RecordInvocation(toolID ToolID, kind string, latency time.Duration, success bool)
	RecordDenial(toolID ToolID, reason string, capability string)
	RecordWorkflowStep(stepID string, kind string, attempts int, success bool, latency time.Duration)
	RecordCacheHit(key string)
	RecordCacheMiss(key string)
}

// NoopMetrics discards every observation.
type NoopMetrics struct{}

func (NoopMetrics) RecordInvocation(ToolID, string, time.Duration, bool)        {}
func (NoopMetrics) RecordDenial(ToolID, string, string)                         {}
func (NoopMetrics) RecordWorkflowStep(string, string, int, bool, time.Duration) {}
func (NoopMetrics) RecordCacheHit(string)                                       {}
func (NoopMetrics) RecordCacheMiss(string)                                      {}

// MetricsSnapshot is an immutable point-in-time copy of MemMetrics.
type MetricsSnapshot struct {
	Invocations          int64            `json:"invocations"`
	Failures             int64            `json:"failures"`
	Denials              int64            `json:"denials"`
	WorkflowSteps        int64            `json:"workflowSteps"`
	WorkflowStepFailures int64            `json:"workflowStepFailures"`
	CacheHits            int64            `json:"cacheHits"`
	CacheMisses          int64            `json:"cacheMisses"`
	AvgLatencyMS         int64            `json:"avgLatencyMs"`
	InvocationsByTool    map[ToolID]int64 `json:"invocationsByTool,omitempty"`
	InvocationsByKind    map[string]int64 `json:"invocationsByKind,omitempty"`
	DenialsByReason      map[string]int64 `json:"denialsByReason,omitempty"`
	DenialsByTool        map[ToolID]int64 `json:"denialsByTool,omitempty"`
	DenialsByCapability  map[string]int64 `json:"denialsByCapability,omitempty"`
	WorkflowStepsByKind  map[string]int64 `json:"workflowStepsByKind,omitempty"`
}

// MemMetrics is a dependency-free in-memory Metrics implementation. External
// exporters can adapt Snapshot or implement Metrics directly.
type MemMetrics struct {
	mu sync.Mutex

	invocations          int64
	failures             int64
	denials              int64
	workflowSteps        int64
	workflowStepFailures int64
	cacheHits            int64
	cacheMisses          int64
	totalLatency         time.Duration

	byTool              map[ToolID]int64
	byKind              map[string]int64
	denialsByReason     map[string]int64
	denialsByTool       map[ToolID]int64
	denialsByCapability map[string]int64
	stepsByKind         map[string]int64
}

// NewMemMetrics returns an empty in-memory metrics collector.
func NewMemMetrics() *MemMetrics {
	return &MemMetrics{
		byTool:              map[ToolID]int64{},
		byKind:              map[string]int64{},
		denialsByReason:     map[string]int64{},
		denialsByTool:       map[ToolID]int64{},
		denialsByCapability: map[string]int64{},
		stepsByKind:         map[string]int64{},
	}
}

// RecordInvocation implements Metrics.
func (m *MemMetrics) RecordInvocation(toolID ToolID, kind string, latency time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invocations++
	if !success {
		m.failures++
	}
	m.totalLatency += latency
	m.byTool[toolID]++
	if kind != "" {
		m.byKind[kind]++
	}
}

// RecordDenial implements Metrics.
func (m *MemMetrics) RecordDenial(toolID ToolID, reason string, capability string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.denials++
	m.denialsByReason[reason]++
	if toolID != "" {
		m.denialsByTool[toolID]++
	}
	if capability != "" {
		m.denialsByCapability[capability]++
	}
}

// RecordWorkflowStep implements Metrics.
func (m *MemMetrics) RecordWorkflowStep(stepID string, kind string, attempts int, success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflowSteps++
	if !success {
		m.workflowStepFailures++
	}
	if kind != "" {
		m.stepsByKind[kind]++
	}
}

// RecordCacheHit implements Metrics.
func (m *MemMetrics) RecordCacheHit(string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheHits++
}

// RecordCacheMiss implements Metrics.
func (m *MemMetrics) RecordCacheMiss(string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheMisses++
}

// Snapshot returns a deep copy of the current counters.
func (m *MemMetrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	avg := int64(0)
	if m.invocations > 0 {
		avg = int64(float64(m.totalLatency) / float64(m.invocations) / float64(time.Millisecond))
	}
	return MetricsSnapshot{
		Invocations:          m.invocations,
		Failures:             m.failures,
		Denials:              m.denials,
		WorkflowSteps:        m.workflowSteps,
		WorkflowStepFailures: m.workflowStepFailures,
		CacheHits:            m.cacheHits,
		CacheMisses:          m.cacheMisses,
		AvgLatencyMS:         avg,
		InvocationsByTool:    copyCounts(m.byTool),
		InvocationsByKind:    copyStringCounts(m.byKind),
		DenialsByReason:      copyStringCounts(m.denialsByReason),
		DenialsByTool:        copyCounts(m.denialsByTool),
		DenialsByCapability:  copyStringCounts(m.denialsByCapability),
		WorkflowStepsByKind:  copyStringCounts(m.stepsByKind),
	}
}

func copyCounts(in map[ToolID]int64) map[ToolID]int64 {
	out := make(map[ToolID]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyStringCounts(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
