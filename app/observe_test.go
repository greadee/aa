package main

import "testing"

func TestControlPlaneSeamsEmitObservations(t *testing.T) {
	g, err := build(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer g.obs.Close()

	b := &backend{g: g}
	if err := b.CreateProject("p1", "Alpha"); err != nil {
		t.Fatal(err)
	}
	events, err := g.obs.Replay("project:p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "project.create" {
		t.Fatalf("project events = %+v", events)
	}

	id, err := b.SubmitWork("default", "first", nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err = g.obs.Replay("project:default", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "work.submit" || events[0].WorkPackageID != id {
		t.Fatalf("work events = %+v", events)
	}

	if err := b.Approve(id); err != nil {
		t.Fatal(err)
	}
	events, err = g.obs.Replay("control-plane", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "gate.approve" {
		t.Fatalf("gate events = %+v", events)
	}
}
