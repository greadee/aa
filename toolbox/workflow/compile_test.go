package workflow

import (
	"errors"
	"reflect"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
)

func wf(id string, steps ...toolbox.WorkflowStep) toolbox.Workflow {
	return toolbox.Workflow{
		Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "workflow", ID: id},
		Name:     id,
		Version:  "1.0",
		Steps:    steps,
	}
}

func step(id, kind string, deps ...string) toolbox.WorkflowStep {
	return toolbox.WorkflowStep{ID: id, Kind: kind, DependsOn: deps}
}

func TestCompileValidWorkflow(t *testing.T) {
	plan, err := Compile(wf("wf_1",
		step("plan", "task"),
		step("implement", "task", "plan"),
		toolbox.WorkflowStep{ID: "verify", Kind: "gate", DependsOn: []string{"implement"}, Gates: []string{"tests"}},
	))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	want := []string{"plan", "implement", "verify"}
	if !reflect.DeepEqual(plan.Order, want) {
		t.Fatalf("order = %v, want %v", plan.Order, want)
	}
	if plan.WorkflowID != "wf_1" || plan.Version != "1.0" {
		t.Fatalf("plan identity = %q %q", plan.WorkflowID, plan.Version)
	}
}

func TestCompileOrderIndependentOfInputOrder(t *testing.T) {
	a := []toolbox.WorkflowStep{
		step("a", "task"),
		step("b", "task"),
		step("c", "task", "a", "b"),
	}
	b := []toolbox.WorkflowStep{a[2], a[1], a[0]}
	p1, err := Compile(wf("wf", a...))
	if err != nil {
		t.Fatalf("Compile a: %v", err)
	}
	p2, err := Compile(wf("wf", b...))
	if err != nil {
		t.Fatalf("Compile b: %v", err)
	}
	if !reflect.DeepEqual(p1.Order, p2.Order) {
		t.Fatalf("order differs: %v vs %v", p1.Order, p2.Order)
	}
	if !reflect.DeepEqual(p1.Order, []string{"a", "b", "c"}) {
		t.Fatalf("order = %v", p1.Order)
	}
}

func TestCompileTieBreakIsLexicographic(t *testing.T) {
	plan, err := Compile(wf("wf", step("z", "task"), step("m", "task"), step("a", "task")))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !reflect.DeepEqual(plan.Order, []string{"a", "m", "z"}) {
		t.Fatalf("order = %v", plan.Order)
	}
}

func TestCompileDetectsCycle(t *testing.T) {
	_, err := Compile(wf("wf", step("a", "task", "b"), step("b", "task", "a")))
	if !errors.Is(err, toolbox.ErrCycle) {
		t.Fatalf("err = %v, want ErrCycle", err)
	}
}

func TestCompileRejectsDanglingDependency(t *testing.T) {
	_, err := Compile(wf("wf", step("a", "task", "ghost")))
	if !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCompileRejectsSelfDependency(t *testing.T) {
	_, err := Compile(wf("wf", step("a", "task", "a")))
	if !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCompileRejectsDuplicateID(t *testing.T) {
	_, err := Compile(wf("wf", step("a", "task"), step("a", "task")))
	if !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCompileRejectsToolStepWithoutTool(t *testing.T) {
	_, err := Compile(wf("wf", step("a", "tool")))
	if !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCompileRejectsGateStepWithoutGates(t *testing.T) {
	_, err := Compile(wf("wf", step("a", "gate")))
	if !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCompileRejectsEmptyWorkflow(t *testing.T) {
	_, err := Compile(wf("wf"))
	if !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCompileNormalizesDependencies(t *testing.T) {
	plan, err := Compile(wf("wf", step("a", "task"), step("b", "task"), step("c", "task", "b", "a", "a")))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !reflect.DeepEqual(plan.Dependencies("c"), []string{"a", "b"}) {
		t.Fatalf("deps = %v", plan.Dependencies("c"))
	}
}
