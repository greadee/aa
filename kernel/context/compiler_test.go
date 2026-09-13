package context

import "testing"

func TestCompileIsOrderIndependent(t *testing.T) {
	inputs := []Input{
		{Kind: "strategy", ID: "str_1", Text: "always write tests"},
		{Kind: "issue", ID: "iss_1", Text: "add the thing"},
	}
	a := Compiler{MaxTokens: 100}.Compile("prj", "wp", inputs)
	reordered := []Input{inputs[1], inputs[0]}
	b := Compiler{MaxTokens: 100}.Compile("prj", "wp", reordered)
	if a.Digest != b.Digest {
		t.Fatalf("digests differ: %s vs %s", a.Digest, b.Digest)
	}
	if len(a.Sections) != 2 || a.Sections[0].Kind != "issue" {
		t.Fatalf("sections = %+v", a.Sections)
	}
	if a.Truncated {
		t.Fatal("should not be truncated")
	}
}

func TestCompileRespectsBudget(t *testing.T) {
	// 400 runes = 100 tokens each; budget 150 tokens fits one and truncates the next.
	long := ""
	for i := 0; i < 400; i++ {
		long += "x"
	}
	inputs := []Input{
		{Kind: "a", ID: "1", Text: long},
		{Kind: "b", ID: "2", Text: long},
	}
	b := Compiler{MaxTokens: 150}.Compile("prj", "wp", inputs)
	if !b.Truncated {
		t.Fatal("expected truncation")
	}
	if b.TokensUsed > 150 {
		t.Fatalf("used %d tokens over budget", b.TokensUsed)
	}
	if len(b.Sections) != 2 {
		t.Fatalf("sections = %d", len(b.Sections))
	}
}

func TestCompileEmpty(t *testing.T) {
	b := Compiler{}.Compile("prj", "wp", nil)
	if b.TokensUsed != 0 || len(b.Sections) != 0 {
		t.Fatalf("bundle = %+v", b)
	}
	if b.TokenBudget != DefaultMaxTokens {
		t.Fatalf("budget = %d", b.TokenBudget)
	}
	if b.Digest == "" {
		t.Fatal("missing digest")
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("empty = %d", got)
	}
	if got := EstimateTokens("abcd"); got != 1 {
		t.Fatalf("abcd = %d", got)
	}
	if got := EstimateTokens("abcde"); got != 2 {
		t.Fatalf("abcde = %d", got)
	}
}
