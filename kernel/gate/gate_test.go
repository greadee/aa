package gate

import "testing"

func TestTestsGate(t *testing.T) {
	if d := (TestsGate{}).Evaluate(Subject{}); d.Passed || d.Reason != "no tests reported" {
		t.Fatalf("decision = %+v", d)
	}
	if d := (TestsGate{}).Evaluate(Subject{Tests: []TestSummary{{Name: "t", Outcome: "failed"}}}); d.Passed {
		t.Fatalf("decision = %+v", d)
	}
	if d := (TestsGate{}).Evaluate(Subject{Tests: []TestSummary{{Name: "t", Outcome: "passed"}}}); !d.Passed {
		t.Fatalf("decision = %+v", d)
	}
}

func TestHumanGatePendingThenApproved(t *testing.T) {
	g := NewHumanGate()
	if d := g.Evaluate(Subject{WorkPackageID: "wp_1"}); !d.Pending || d.Passed {
		t.Fatalf("decision = %+v", d)
	}
	g.Approve("wp_1")
	if d := g.Evaluate(Subject{WorkPackageID: "wp_1"}); !d.Passed {
		t.Fatalf("decision = %+v", d)
	}
}

func TestEvaluateComposite(t *testing.T) {
	gates := []Gate{TestsGate{}, NewHumanGate()}
	report := Evaluate(gates, Subject{WorkPackageID: "wp_1", Tests: []TestSummary{{Name: "t", Outcome: "passed"}}})
	if report.Passed || !report.Pending {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Decisions) != 2 || report.Decisions[0].Gate != "human_review" {
		t.Fatalf("decisions = %+v", report.Decisions)
	}

	human := NewHumanGate()
	human.Approve("wp_1")
	report = Evaluate([]Gate{TestsGate{}, human}, Subject{WorkPackageID: "wp_1", Tests: []TestSummary{{Name: "t", Outcome: "passed"}}})
	if !report.Passed || report.Pending {
		t.Fatalf("report = %+v", report)
	}
}

func TestEvaluateFailure(t *testing.T) {
	report := Evaluate([]Gate{TestsGate{}}, Subject{Tests: []TestSummary{{Name: "t", Outcome: "failed"}}})
	if report.Passed || report.Pending {
		t.Fatalf("report = %+v", report)
	}
}
