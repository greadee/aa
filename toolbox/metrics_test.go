package toolbox

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMemMetricsZeroValueIsUsable(t *testing.T) {
	var m MemMetrics
	m.RecordInvocation("tool", "tool", 0, true)
	m.RecordRPC("method", 0, false)
	m.RecordRPCError("method", "invalid")
	snap := m.Snapshot()
	if snap.Invocations != 1 || snap.RPCRequests != 1 || snap.RPCErrors != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestMetricsSnapshotWireKeys(t *testing.T) {
	snap := MetricsSnapshot{Invocations: 3, Denials: 2, AvgLatencyMS: 7}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"invocations", "denials", "avgLatencyMs"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("snapshot JSON missing %q: %s", key, data)
		}
	}
	for _, old := range []string{"Invocations", "Denials", "AvgLatencyMS"} {
		if _, ok := got[old]; ok {
			t.Fatalf("snapshot JSON retained legacy key %q: %s", old, data)
		}
	}
}

func TestMetricsInterfaceCoversMemMetrics(t *testing.T) {
	var _ Metrics = (*MemMetrics)(nil)
}

func TestMemMetricsRecordsLabels(t *testing.T) {
	var m MemMetrics
	m.RecordInvocation("tool_a", "tool", 10, false)
	m.RecordDenial("tool_a", "missing_capability", "write_workspace")
	m.RecordCacheMiss("key")
	m.RecordWorkflowStep("step_1", "tool", 2, false, 5)
	snap := m.Snapshot()
	if snap.Invocations != 1 || snap.Failures != 1 || snap.Denials != 1 || snap.CacheMisses != 1 || snap.WorkflowStepFailures != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
	if snap.DenialsByReason["missing_capability"] != 1 || snap.DenialsByCapability["write_workspace"] != 1 {
		t.Fatalf("denial labels = %+v %+v", snap.DenialsByReason, snap.DenialsByCapability)
	}
	if snap.WorkflowStepsByKind["tool"] != 1 || snap.WorkflowAttempts != 2 {
		t.Fatalf("workflow labels = %+v attempts=%d", snap.WorkflowStepsByKind, snap.WorkflowAttempts)
	}
}

func TestLogMetricsExporter(t *testing.T) {
	var buf bytes.Buffer
	m := NewLogMetrics(&buf)
	m.RecordInvocation("tool_a", "tool", 12, false)
	m.RecordDenial("tool_a", "missing_capability", "write_workspace")
	m.RecordWorkflowStep("step_1", "tool", 2, false, 5)
	m.RecordCacheHit("key")
	m.RecordRPCError("toolbox.invoke", "invalid")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d: %s", len(lines), buf.String())
	}
	wantEvents := []string{"toolbox.invocation", "toolbox.denial", "toolbox.workflow_step", "toolbox.cache_hit", "toolbox.rpc_error"}
	for i, line := range lines {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		if event["event"] != wantEvents[i] {
			t.Fatalf("line %d event = %v, want %s", i, event["event"], wantEvents[i])
		}
	}
}
