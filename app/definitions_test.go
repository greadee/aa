package main

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/greadee/aa/app/internal/controlplane"
)

func TestDefinitionsPersistAcrossBuild(t *testing.T) {
	root := t.TempDir()
	g, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	d := definitionsBackend{g: g}

	if err := d.AddTeam(json.RawMessage(`{"contractVersion":"2.0","kind":"team_spec","id":"team_engineering","title":"Engineering","roles":["Builder","Inspector"]}`)); err != nil {
		t.Fatalf("add team: %v", err)
	}
	if err := d.AddRoutine(json.RawMessage(`{"contractVersion":"2.0","kind":"routine","id":"routine_feature","name":"Feature Delivery","version":"1.0","stages":[{"id":"s1","title":"Implement"}]}`)); err != nil {
		t.Fatalf("add routine: %v", err)
	}
	if err := d.AddPolicy(json.RawMessage(`{"contractVersion":"2.0","kind":"policy_definition","id":"policy_budget","title":"Budget","policyKind":"budget","version":"1.0"}`)); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if err := g.obs.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.obs.Close()
	if _, ok := reopened.teams.Get("team_engineering"); !ok {
		t.Fatal("team definition did not persist")
	}
	if _, ok := reopened.routines.Get("routine_feature"); !ok {
		t.Fatal("routine definition did not persist")
	}
	if _, ok := reopened.policies.Get("policy_budget"); !ok {
		t.Fatal("policy definition did not persist")
	}
}

func TestDuplicateDefinitionIsConflict(t *testing.T) {
	g, err := build(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer g.obs.Close()
	d := definitionsBackend{g: g}
	spec := json.RawMessage(`{"contractVersion":"2.0","kind":"team_spec","id":"team_engineering","title":"Engineering"}`)
	if err := d.AddTeam(spec); err != nil {
		t.Fatal(err)
	}
	err = d.AddTeam(spec)
	if !errors.Is(err, controlplane.ErrDefinitionConflict) {
		t.Fatalf("err = %v, want ErrDefinitionConflict", err)
	}
}

func TestInvalidDefinitionIsRejected(t *testing.T) {
	g, err := build(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer g.obs.Close()
	d := definitionsBackend{g: g}
	if err := d.AddTeam(json.RawMessage(`{"kind":"team_spec","id":"team_bad"}`)); err == nil {
		t.Fatal("expected a validation error")
	}
}
