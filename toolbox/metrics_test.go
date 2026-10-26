package toolbox

import "testing"

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
