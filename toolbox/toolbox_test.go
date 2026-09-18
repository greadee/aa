package toolbox

import "testing"

func TestKeyIsStableAndTrimmed(t *testing.T) {
	got := Key(" invoke ", " a ", "b")
	if got != "invoke:a:b" {
		t.Fatalf("Key = %q, want %q", got, "invoke:a:b")
	}
	if again := Key("invoke", "a", "b"); again != got {
		t.Fatalf("Key not stable: %q vs %q", again, got)
	}
}

func TestInvocationKeyNamespacesExplicitByTool(t *testing.T) {
	got := InvocationKey("tool_1", "ec_1", "explicit")
	if got != "invoke:tool_1:explicit" {
		t.Fatalf("explicit key = %q, want %q", got, "invoke:tool_1:explicit")
	}
	if other := InvocationKey("tool_2", "ec_1", "explicit"); other == got {
		t.Fatalf("explicit keys should differ across tools: %q", other)
	}
}

func TestInvocationKeyDerivesWhenEmpty(t *testing.T) {
	got := InvocationKey("tool_1", "ec_1", "")
	if got != "invoke:tool_1:ec_1" {
		t.Fatalf("derived key = %q, want %q", got, "invoke:tool_1:ec_1")
	}
	if same := InvocationKey("tool_1", "ec_1", ""); same != got {
		t.Fatalf("derived key not stable: %q vs %q", same, got)
	}
	if other := InvocationKey("tool_2", "ec_1", ""); other == got {
		t.Fatalf("derived keys should differ across tools: %q", other)
	}
}

func TestFixedClockAdvancesByStep(t *testing.T) {
	c := NewFixedClock()
	first := c.Now()
	second := c.Now()
	if !second.After(first) {
		t.Fatalf("clock did not advance: %v then %v", first, second)
	}
	if second.Sub(first) != 1e9 {
		t.Fatalf("step = %v, want 1s", second.Sub(first))
	}
}

func TestHashIsStable(t *testing.T) {
	a := HashString("hello")
	b := Hash([]byte("hello"))
	if a != b {
		t.Fatalf("Hash and HashString differ: %q vs %q", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("hash length = %d, want 64", len(a))
	}
}
