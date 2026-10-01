package observability

import "testing"

func identity(worker string) Identity {
	return Identity{
		Organization: "acme", Project: "aa", Subtask: "wp_1", Crew: "crew_a",
		Worker: worker, Role: "Builder", Team: "site", Model: "ollama/qwen",
	}
}

func TestIdentityChainAndValidation(t *testing.T) {
	i := identity("w1")
	if got := i.Chain(); got != "acme/aa/wp_1/crew_a/w1/Builder/site/ollama/qwen" {
		t.Fatalf("chain = %q", got)
	}
	if err := i.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := i
	bad.Subtask = ""
	bad.Worker = ""
	bad.Role = ""
	if err := bad.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSummarizeAndAccounting(t *testing.T) {
	rec := NewMemRecorder()
	events := []Event{
		{Name: RuntimeStart, CorrelationID: "c1", Identity: identity("w1"), Tokens: 100, CostUSD: 0.01, ComputeMS: 20},
		{Name: FailureRetry, CorrelationID: "c1", Identity: identity("w1"), Retry: true, Failed: true},
		{Name: AgentResult, CorrelationID: "c2", Identity: identity("w2"), Tokens: 50, CostUSD: 0.02, ComputeMS: 30},
	}
	for _, event := range events {
		if err := rec.Record(event); err != nil {
			t.Fatal(err)
		}
	}
	m := Summarize(rec.Events())
	if m.Events != 3 || m.Failed != 1 || m.Retries != 1 || m.Tokens != 150 || m.ComputeMS != 50 {
		t.Fatalf("metrics = %+v", m)
	}
	accounts := Accounting(rec.Events())
	if len(accounts) != 2 || accounts[0].Chain != identity("w1").Chain() {
		t.Fatalf("accounts = %+v", accounts)
	}
	if accounts[0].Metrics.Tokens != 100 || accounts[0].Metrics.Failed != 1 {
		t.Fatalf("w1 account = %+v", accounts[0])
	}
}

func TestRecordRejectsInvalid(t *testing.T) {
	rec := NewMemRecorder()
	if err := rec.Record(Event{Name: RuntimeStart, Identity: Identity{Project: "aa"}}); err == nil {
		t.Fatal("expected validation error")
	}
	if err := rec.Record(Event{Identity: identity("w1")}); err == nil {
		t.Fatal("expected name error")
	}
}
