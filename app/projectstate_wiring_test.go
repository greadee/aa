package main

import "testing"

func TestControlPlaneProjectsAndWorkAreWired(t *testing.T) {
	root := t.TempDir()
	g, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	defer g.obs.Close()
	b := &backend{g: g}

	if _, err := b.SubmitWork("default", "default work", nil); err != nil {
		t.Fatalf("default submit: %v", err)
	}
	if err := b.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	id, err := b.SubmitWork("p1", "project work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty work id")
	}
	work := b.WorkPackages("p1")
	if len(work) != 1 || work[0].ID != id {
		t.Fatalf("work = %+v", work)
	}
	if _, err := b.SubmitWork("missing", "work", nil); err == nil {
		t.Fatal("expected unknown project error")
	}
	if err := b.Approve("missing"); err == nil {
		t.Fatal("expected unknown gate error")
	}
	if len(b.Projects()) != 2 {
		t.Fatalf("projects = %+v", b.Projects())
	}
}

func TestControlPlaneProjectWorkSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	g, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	b := &backend{g: g}
	if err := b.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	id, err := b.SubmitWork("p1", "project work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.obs.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.obs.Close()
	b2 := &backend{g: reopened}
	work := b2.WorkPackages("p1")
	if len(work) != 1 || work[0].ID != id {
		t.Fatalf("work after restart = %+v", work)
	}
	// The default project is seeded idempotently, not duplicated.
	if len(b2.Projects()) != 2 {
		t.Fatalf("projects after restart = %+v", b2.Projects())
	}
}

func TestControlPlaneApprovalSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	g, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	b := &backend{g: g}
	if err := b.CreateProject("p1", "Project One"); err != nil {
		t.Fatal(err)
	}
	id, err := b.SubmitWork("p1", "project work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Approve(id); err != nil {
		t.Fatal(err)
	}
	if err := g.obs.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.obs.Close()
	approvals, err := reopened.projects.ApprovalRecords()
	if err != nil || len(approvals) != 1 || approvals[0].Decision != "approve" {
		t.Fatalf("approvals = %+v err = %v", approvals, err)
	}
	if reopened.projects.ActiveAssignments() != 0 {
		t.Fatalf("active assignments = %d", reopened.projects.ActiveAssignments())
	}
}
