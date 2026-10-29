package projectstate

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/gate"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

func newState(t *testing.T) (*State, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state, err := New(st, projection.NewMem())
	if err != nil {
		t.Fatal(err)
	}
	return state, st
}

func seedAwaitingGate(t *testing.T, st *store.Store, projectID, workPackageID string) {
	t.Helper()
	assignments := repo.NewAssignmentRepository(st, projection.NewMem())
	if _, err := assignments.Save(repo.AssignmentRecord{
		ID: "asg_" + workPackageID, ProjectID: projectID, WorkPackageID: workPackageID,
		WorkerID: "w1", State: "awaiting_gates", Attempt: 1,
	}); err != nil {
		t.Fatal(err)
	}
}
func TestCreateProjectAndSubmitScopedWork(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	if _, err := state.CreateProject("p2", "Project Two"); err != nil {
		t.Fatal(err)
	}
	first, err := state.SubmitWork("p1", "First", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := state.SubmitWork("p1", "Second", []string{first.ID})
	if err != nil {
		t.Fatal(err)
	}
	other, err := state.SubmitWork("p2", "Other", nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.ID == other.ID {
		t.Fatalf("work ids are not unique: %q %q %q", first.ID, second.ID, other.ID)
	}
	p1Work, err := state.WorkPackages("p1")
	if err != nil || len(p1Work) != 2 {
		t.Fatalf("p1 work = %v err = %v", p1Work, err)
	}
	p2Work, err := state.WorkPackages("p2")
	if err != nil || len(p2Work) != 1 || p2Work[0].ID != other.ID {
		t.Fatalf("p2 work = %v err = %v", p2Work, err)
	}
	if _, err := state.WorkPackages("missing"); err == nil {
		t.Fatal("expected unknown project error")
	}
}

func TestReopenRestoresProjectsAndWork(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Persisted", nil)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(st, projection.NewMem())
	if err != nil {
		t.Fatal(err)
	}
	projects := reopened.Projects()
	if len(projects) != 1 || projects[0].ID != "p1" {
		t.Fatalf("projects = %+v", projects)
	}
	workAfter, err := reopened.WorkPackages("p1")
	if err != nil || len(workAfter) != 1 || workAfter[0].ID != work.ID {
		t.Fatalf("work = %+v err = %v", workAfter, err)
	}
}

func TestDuplicateAndUnknownProjectFailClosed(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	if _, err := state.CreateProject("p1", "Duplicate"); !errors.Is(err, repo.ErrExists) {
		t.Fatalf("duplicate err = %v, want ErrExists", err)
	}
	if _, err := state.SubmitWork("missing", "Work", nil); err == nil {
		t.Fatal("expected unknown project error")
	}
}

func TestSubmitWorkRejectsUnknownDependency(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	if _, err := state.SubmitWork("p1", "Work", []string{"ghost"}); err == nil {
		t.Fatal("expected unknown dependency error")
	}
}

func TestDefaultAcceptanceAndContractState(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(work.Acceptance) == 0 {
		t.Fatal("acceptance was not defaulted")
	}
	sched, _, ok := state.ProjectRuntime("p1")
	if !ok {
		t.Fatal("runtime missing")
	}
	status := sched.Status()
	if len(status.WorkPackages) != 1 || status.WorkPackages[0].State != "PROPOSED" {
		t.Fatalf("scheduler status = %+v", status)
	}
}

func TestProjectRecordStateIsCanonical(t *testing.T) {
	state, _ := newState(t)
	project, err := state.CreateProject("p1", "Project One")
	if err != nil {
		t.Fatal(err)
	}
	if project.Kind != "project_record" || project.State != v2.ProjectCreated || project.LayoutVersion != "1" {
		t.Fatalf("project = %+v", project)
	}
}

func TestEnsureProjectIsIdempotent(t *testing.T) {
	state, _ := newState(t)
	first, created, err := state.EnsureProject("default", "Default")
	if err != nil || !created {
		t.Fatalf("first ensure = %+v created=%v err=%v", first, created, err)
	}
	second, created, err := state.EnsureProject("default", "Default")
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("second ensure = %+v created=%v err=%v", second, created, err)
	}
	if len(state.Projects()) != 1 {
		t.Fatalf("projects = %+v", state.Projects())
	}
}

func TestProjectForWorkPackageAndAssignments(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	project, ok := state.ProjectForWorkPackage(work.ID)
	if !ok || project.Record.ID != "p1" {
		t.Fatalf("project = %+v ok=%v", project, ok)
	}
	if _, ok := state.ProjectForWorkPackage("missing"); ok {
		t.Fatal("unexpected project for missing work")
	}
	if len(state.AssignmentStatuses()) != 0 {
		t.Fatalf("assignments = %+v", state.AssignmentStatuses())
	}
	if state.ActiveAssignments() != 0 {
		t.Fatalf("active = %d", state.ActiveAssignments())
	}
}

func TestRecordApprovalPersistsAndRestores(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	seedAwaitingGate(t, st, "p1", work.ID)
	if _, err := state.RecordApproval("p1", work.ID, repo.DecisionApprove, "human", "ok"); err != nil {
		t.Fatal(err)
	}
	records, err := state.ApprovalRecords()
	if err != nil || len(records) != 1 || records[0].Decision != repo.DecisionApprove {
		t.Fatalf("approvals = %+v err = %v", records, err)
	}
	reopened, err := New(st, projection.NewMem())
	if err != nil {
		t.Fatal(err)
	}
	records, err = reopened.ApprovalRecords()
	if err != nil || len(records) != 1 || records[0].Decision != repo.DecisionApprove {
		t.Fatalf("reopened approvals = %+v err = %v", records, err)
	}
	if _, err := state.RecordApproval("p1", "missing", repo.DecisionApprove, "human", ""); err == nil {
		t.Fatal("expected unknown work package error")
	}
	if _, err := state.RecordApproval("missing", work.ID, repo.DecisionApprove, "human", ""); err == nil {
		t.Fatal("expected unknown project error")
	}
}

func TestDurableAssignmentReadModel(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	assignments := repo.NewAssignmentRepository(st, projection.NewMem())
	record := repo.AssignmentRecord{
		ID:            "asg_1",
		ProjectID:     "p1",
		WorkPackageID: work.ID,
		WorkerID:      "w1",
		State:         "running",
		Attempt:       1,
	}
	if _, err := assignments.Save(record); err != nil {
		t.Fatal(err)
	}
	statuses := state.AssignmentStatuses()
	if len(statuses) != 1 || statuses[0].State != "running" || state.ActiveAssignments() != 1 {
		t.Fatalf("statuses = %+v active = %d", statuses, state.ActiveAssignments())
	}
	record.State = "accepted"
	if _, err := assignments.Save(record); err != nil {
		t.Fatal(err)
	}
	if state.ActiveAssignments() != 0 {
		t.Fatalf("active after accepted = %d", state.ActiveAssignments())
	}
	reopened, err := New(st, projection.NewMem())
	if err != nil {
		t.Fatal(err)
	}
	if statuses := reopened.AssignmentStatuses(); len(statuses) != 1 || statuses[0].State != "accepted" {
		t.Fatalf("reopened statuses = %+v", statuses)
	}
}

func TestNewRejectsAssignmentWithUnknownWorkPackage(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	assignments := repo.NewAssignmentRepository(st, projection.NewMem())
	if _, err := assignments.Save(repo.AssignmentRecord{
		ID:            "asg_bad",
		ProjectID:     "p1",
		WorkPackageID: "missing",
		WorkerID:      "w1",
		State:         "running",
		Attempt:       1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := New(st, projection.NewMem()); err == nil {
		t.Fatal("expected corrupt assignment state to fail closed")
	}
}

func TestApprovalSemanticsApproveRejectAndConflict(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	project, ok := state.ProjectForWorkPackage(work.ID)
	if !ok {
		t.Fatal("project missing")
	}
	seedAwaitingGate(t, st, "p1", work.ID)
	if _, err := state.RecordApproval("p1", work.ID, repo.DecisionApprove, "human", "ok"); err != nil {
		t.Fatal(err)
	}
	decision := project.Gate.Evaluate(gate.Subject{WorkPackageID: work.ID})
	if !decision.Passed || decision.Pending {
		t.Fatalf("gate = %+v", decision)
	}
	if _, err := state.RecordApproval("p1", work.ID, repo.DecisionApprove, "human", "again"); !errors.Is(err, ErrApprovalResolved) {
		t.Fatalf("duplicate err = %v, want ErrApprovalResolved", err)
	}

	second, err := state.SubmitWork("p1", "Second", nil)
	if err != nil {
		t.Fatal(err)
	}
	seedAwaitingGate(t, st, "p1", second.ID)
	if _, err := state.RecordApproval("p1", second.ID, repo.DecisionReject, "human", "no"); err != nil {
		t.Fatal(err)
	}
	decision = project.Gate.Evaluate(gate.Subject{WorkPackageID: second.ID})
	if decision.Passed || decision.Pending {
		t.Fatalf("rejected gate = %+v", decision)
	}
	if _, err := state.RecordApproval("p1", second.ID, repo.DecisionApprove, "human", ""); !errors.Is(err, ErrApprovalResolved) {
		t.Fatalf("resolved err = %v, want ErrApprovalResolved", err)
	}

	// Terminal assignment state also resolves the gate.
	third, err := state.SubmitWork("p1", "Third", nil)
	if err != nil {
		t.Fatal(err)
	}
	assignments := repo.NewAssignmentRepository(st, projection.NewMem())
	if _, err := assignments.Save(repo.AssignmentRecord{
		ID: "asg_third", ProjectID: "p1", WorkPackageID: third.ID, WorkerID: "w1", State: "accepted", Attempt: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := state.RecordApproval("p1", third.ID, repo.DecisionApprove, "human", ""); !errors.Is(err, ErrApprovalResolved) {
		t.Fatalf("terminal err = %v, want ErrApprovalResolved", err)
	}
}

func TestApprovalValidation(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.RecordApproval("p1", work.ID, "bogus", "human", ""); !errors.Is(err, ErrInvalidApproval) {
		t.Fatalf("decision err = %v, want ErrInvalidApproval", err)
	}
	if _, err := state.RecordApproval("p1", work.ID, repo.DecisionApprove, "", ""); !errors.Is(err, ErrInvalidApproval) {
		t.Fatalf("actor err = %v, want ErrInvalidApproval", err)
	}
	if _, err := state.RecordApproval("p1", "missing", repo.DecisionApprove, "human", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("work err = %v, want ErrNotFound", err)
	}
	if _, err := state.RecordApproval("missing", work.ID, repo.DecisionApprove, "human", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("project err = %v, want ErrNotFound", err)
	}
}

func TestTaskGraphReadinessAndWorkPackageLookup(t *testing.T) {
	state, _ := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	empty, ok := state.Graph("p1")
	if !ok || empty.Kind != "task_graph" || len(empty.Nodes) != 0 {
		t.Fatalf("empty graph = %+v ok=%v", empty, ok)
	}
	first, err := state.SubmitWork("p1", "First", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := state.SubmitWork("p1", "Second", []string{first.ID})
	if err != nil {
		t.Fatal(err)
	}
	graph, ok := state.Graph("p1")
	if !ok || len(graph.Nodes) != 2 {
		t.Fatalf("graph = %+v ok=%v", graph, ok)
	}
	readiness := map[string]string{}
	for _, node := range graph.Nodes {
		readiness[node.WorkPackageID] = string(node.Readiness)
	}
	if readiness[first.ID] != "ready" || readiness[second.ID] != "blocked" {
		t.Fatalf("readiness = %+v", readiness)
	}
	project, _ := state.ProjectForWorkPackage(first.ID)
	if err := project.Graph.SetState(first.ID, planner.StateCompleted); err != nil {
		t.Fatal(err)
	}
	graph, _ = state.Graph("p1")
	readiness = map[string]string{}
	for _, node := range graph.Nodes {
		readiness[node.WorkPackageID] = string(node.Readiness)
	}
	if readiness[first.ID] != "satisfied" || readiness[second.ID] != "ready" {
		t.Fatalf("readiness after completion = %+v", readiness)
	}
	work, ok := state.WorkPackage("p1", second.ID)
	if !ok || work.ID != second.ID {
		t.Fatalf("work package = %+v ok=%v", work, ok)
	}
	if _, ok := state.WorkPackage("p2", second.ID); ok {
		t.Fatal("work package leaked across projects")
	}
	if _, ok := state.Graph("missing"); ok {
		t.Fatal("unexpected graph for missing project")
	}
}

func TestNewRejectsUnknownWorkPackageState(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProject(t, st, "p1"); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"contractVersion":"2.0","kind":"work_package","id":"wp_bad","projectId":"p1","revision":1,"title":"bad","state":"BOGUS","acceptance":["done"]}`)
	rec, err := store.NewRecord("work_package", "wp_bad", 1, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.PutRecord(rec); err != nil {
		t.Fatal(err)
	}
	if _, err := New(st, projection.NewMem()); err == nil {
		t.Fatal("expected unknown work package state to fail closed")
	}
}

func TestVerifyingWorkPackageStateIsNotRemappedToProposed(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	workRepo := repo.NewWorkPackageRepository(st, projection.NewMem())
	if _, err := workRepo.UpdateState(work.ID, v2.WorkPackageVerifying); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(st, projection.NewMem())
	if err != nil {
		t.Fatal(err)
	}
	graph, ok := reopened.Graph("p1")
	if !ok || len(graph.Nodes) != 1 || graph.Nodes[0].Readiness != v2.ReadinessDispatched {
		t.Fatalf("graph = %+v ok=%v", graph, ok)
	}
}

func writeProject(t *testing.T, st *store.Store, id string) error {
	t.Helper()
	data := []byte(`{"contractVersion":"2.0","kind":"project_record","id":"` + id + `","revision":1,"name":"` + id + `","state":"CREATED","layoutVersion":"1"}`)
	rec, err := store.NewRecord("project_record", id, 1, data)
	if err != nil {
		return err
	}
	_, err = st.PutRecord(rec)
	return err
}

func TestApprovalRequiresPendingGate(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.RecordApproval("p1", work.ID, repo.DecisionApprove, "human", ""); !errors.Is(err, ErrApprovalResolved) {
		t.Fatalf("no assignment err = %v, want ErrApprovalResolved", err)
	}
	assignments := repo.NewAssignmentRepository(st, projection.NewMem())
	if _, err := assignments.Save(repo.AssignmentRecord{
		ID: "asg_running", ProjectID: "p1", WorkPackageID: work.ID,
		WorkerID: "w1", State: "running", Attempt: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := state.RecordApproval("p1", work.ID, repo.DecisionApprove, "human", ""); !errors.Is(err, ErrApprovalResolved) {
		t.Fatalf("running assignment err = %v, want ErrApprovalResolved", err)
	}
}

func TestConcurrentApprovalDecisionsAreSerialized(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	seedAwaitingGate(t, st, "p1", work.ID)

	decisions := []string{repo.DecisionApprove, repo.DecisionReject}
	errs := make([]error, len(decisions))
	var wg sync.WaitGroup
	for i, decision := range decisions {
		wg.Add(1)
		go func(i int, decision string) {
			defer wg.Done()
			_, errs[i] = state.RecordApproval("p1", work.ID, decision, "human", "")
		}(i, decision)
	}
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrApprovalResolved) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want exactly 1 (errs=%v)", successes, errs)
	}
	records, err := state.ApprovalRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("approvals = %+v err = %v", records, err)
	}
	project, ok := state.ProjectForWorkPackage(work.ID)
	if !ok {
		t.Fatal("project missing")
	}
	decision := project.Gate.Evaluate(gate.Subject{WorkPackageID: work.ID})
	if records[0].Decision == repo.DecisionApprove && !decision.Passed {
		t.Fatalf("durable approval and gate disagree: %+v", decision)
	}
	if records[0].Decision == repo.DecisionReject && decision.Passed {
		t.Fatalf("durable rejection and gate disagree: %+v", decision)
	}
}

func TestApprovalAndGateAgreeAfterRestart(t *testing.T) {
	for _, want := range []string{repo.DecisionApprove, repo.DecisionReject} {
		t.Run(want, func(t *testing.T) {
			state, st := newState(t)
			if _, err := state.CreateProject("p1", "Project One"); err != nil {
				t.Fatal(err)
			}
			work, err := state.SubmitWork("p1", "Work", nil)
			if err != nil {
				t.Fatal(err)
			}
			seedAwaitingGate(t, st, "p1", work.ID)
			if _, err := state.RecordApproval("p1", work.ID, want, "human", ""); err != nil {
				t.Fatal(err)
			}
			reopened, err := New(st, projection.NewMem())
			if err != nil {
				t.Fatal(err)
			}
			project, ok := reopened.ProjectForWorkPackage(work.ID)
			if !ok {
				t.Fatal("project missing after restart")
			}
			decision := project.Gate.Evaluate(gate.Subject{WorkPackageID: work.ID})
			if want == repo.DecisionApprove && !decision.Passed {
				t.Fatalf("approval did not survive restart: %+v", decision)
			}
			if want == repo.DecisionReject && decision.Passed {
				t.Fatalf("rejection did not survive restart: %+v", decision)
			}
		})
	}
}

func TestReopenReconcilesExpiredAssignments(t *testing.T) {
	state, st := newState(t)
	if _, err := state.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	work, err := state.SubmitWork("p1", "Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	assignments := repo.NewAssignmentRepository(st, projection.NewMem())
	if _, err := assignments.Save(repo.AssignmentRecord{
		ID: "asg_wp_1", ProjectID: "p1", WorkPackageID: work.ID, WorkerID: "w1",
		State: "leased", Attempt: 1, LeaseOwner: "w1",
		LeaseExpiresAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(st, projection.NewMem())
	if err != nil {
		t.Fatal(err)
	}
	statuses := reopened.AssignmentStatuses()
	if len(statuses) != 1 || statuses[0].State != "expired" {
		t.Fatalf("statuses = %+v, want expired", statuses)
	}
	persisted, err := assignments.Get("asg_wp_1")
	if err != nil || persisted.State != "expired" {
		t.Fatalf("persisted = %+v err=%v", persisted, err)
	}
}
