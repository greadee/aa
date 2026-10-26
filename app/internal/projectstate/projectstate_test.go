package projectstate

import (
	"errors"
	"testing"

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
