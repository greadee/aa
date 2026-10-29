package toolbox

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// LogMetrics emits one structured JSON object per observation. It is a
// dependency-free external exporter: pipe it to a log shipper, Promtail, or any
// collector that understands JSON lines.
type LogMetrics struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// NewLogMetrics returns a structured-log exporter writing to w.
func NewLogMetrics(w io.Writer) *LogMetrics {
	return &LogMetrics{enc: json.NewEncoder(w)}
}

func (m *LogMetrics) emit(event string, fields map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fields["event"] = event
	_ = m.enc.Encode(fields)
}

// RecordInvocation implements Metrics.
func (m *LogMetrics) RecordInvocation(toolID ToolID, kind string, latency time.Duration, success bool) {
	m.emit("toolbox.invocation", map[string]any{
		"toolId":    toolID,
		"kind":      kind,
		"latencyMs": latency.Milliseconds(),
		"success":   success,
	})
}

// RecordDenial implements Metrics.
func (m *LogMetrics) RecordDenial(toolID ToolID, reason string, capability string) {
	m.emit("toolbox.denial", map[string]any{
		"toolId":     toolID,
		"reason":     reason,
		"capability": capability,
	})
}

// RecordWorkflowStep implements Metrics.
func (m *LogMetrics) RecordWorkflowStep(stepID string, kind string, attempts int, success bool, latency time.Duration) {
	m.emit("toolbox.workflow_step", map[string]any{
		"stepId":    stepID,
		"kind":      kind,
		"attempts":  attempts,
		"success":   success,
		"latencyMs": latency.Milliseconds(),
	})
}

// RecordCacheHit implements Metrics.
func (m *LogMetrics) RecordCacheHit(key string) {
	m.emit("toolbox.cache_hit", map[string]any{"key": key})
}

// RecordCacheMiss implements Metrics.
func (m *LogMetrics) RecordCacheMiss(key string) {
	m.emit("toolbox.cache_miss", map[string]any{"key": key})
}

// RecordRPC implements Metrics.
func (m *LogMetrics) RecordRPC(method string, latency time.Duration, success bool) {
	m.emit("toolbox.rpc", map[string]any{
		"method":    method,
		"latencyMs": latency.Milliseconds(),
		"success":   success,
	})
}

// RecordRPCError implements Metrics.
func (m *LogMetrics) RecordRPCError(method string, category string) {
	m.emit("toolbox.rpc_error", map[string]any{
		"method":   method,
		"category": category,
	})
}
