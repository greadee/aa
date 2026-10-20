package main

import (
	"encoding/json"
	"fmt"

	"github.com/greadee/aa/app/internal/controlplane"
	"github.com/greadee/aa/memory/store"
	"github.com/greadee/aa/registry/policies"
	"github.com/greadee/aa/registry/routines"
	"github.com/greadee/aa/registry/teams"
)

// Definition record kinds in the canonical memory store.
const (
	kindTeamSpec   = "team_spec"
	kindRoutine    = "routine"
	kindPolicySpec = "policy_definition"
)

// definitionsBackend serves the durable definition stores and persists each
// accepted definition through the memory store.
type definitionsBackend struct {
	g *runtime
}

func (d definitionsBackend) Teams() []json.RawMessage {
	return marshalSpecs(d.g.teams.List())
}

func (d definitionsBackend) Team(id string) (json.RawMessage, bool) {
	spec, ok := d.g.teams.Get(id)
	if !ok {
		return nil, false
	}
	return marshalSpec(spec)
}

func (d definitionsBackend) AddTeam(raw json.RawMessage) error {
	var spec teams.TeamSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("invalid team spec: %w", err)
	}
	if _, ok := d.g.teams.Get(spec.ID); ok {
		return fmt.Errorf("%w: team %q", controlplane.ErrDefinitionConflict, spec.ID)
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	if err := d.persist(kindTeamSpec, spec.ID, spec); err != nil {
		return err
	}
	return d.g.teams.Add(spec)
}

func (d definitionsBackend) Routines() []json.RawMessage {
	return marshalSpecs(d.g.routines.List())
}

func (d definitionsBackend) Routine(id string) (json.RawMessage, bool) {
	spec, ok := d.g.routines.Get(id)
	if !ok {
		return nil, false
	}
	return marshalSpec(spec)
}

func (d definitionsBackend) AddRoutine(raw json.RawMessage) error {
	var spec routines.RoutineSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("invalid routine spec: %w", err)
	}
	if _, ok := d.g.routines.Get(spec.ID); ok {
		return fmt.Errorf("%w: routine %q", controlplane.ErrDefinitionConflict, spec.ID)
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	if err := d.persist(kindRoutine, spec.ID, spec); err != nil {
		return err
	}
	return d.g.routines.Add(spec)
}

func (d definitionsBackend) Policies() []json.RawMessage {
	return marshalSpecs(d.g.policies.List())
}

func (d definitionsBackend) Policy(id string) (json.RawMessage, bool) {
	spec, ok := d.g.policies.Get(id)
	if !ok {
		return nil, false
	}
	return marshalSpec(spec)
}

func (d definitionsBackend) AddPolicy(raw json.RawMessage) error {
	var spec policies.PolicySpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("invalid policy spec: %w", err)
	}
	if _, ok := d.g.policies.Get(spec.ID); ok {
		return fmt.Errorf("%w: policy %q", controlplane.ErrDefinitionConflict, spec.ID)
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	if err := d.persist(kindPolicySpec, spec.ID, spec); err != nil {
		return err
	}
	return d.g.policies.Add(spec)
}

func (d definitionsBackend) persist(kind, id string, spec any) error {
	data, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	rec, err := store.NewRecord(kind, id, 1, data)
	if err != nil {
		return err
	}
	_, err = d.g.store.PutRecord(rec)
	return err
}

func marshalSpec(spec any) (json.RawMessage, bool) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, false
	}
	return data, true
}

func marshalSpecs[T any](specs []T) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(specs))
	for _, spec := range specs {
		if data, ok := marshalSpec(spec); ok {
			out = append(out, data)
		}
	}
	return out
}

// loadDefinitions restores the durable definition registries from the memory
// store. Corruption fails the build rather than serving a partial catalogue.
func loadDefinitions(st *store.Store, teamsReg *teams.Registry, routinesReg *routines.Registry, policiesReg *policies.Registry) error {
	teamRecords, err := st.ListRecords(kindTeamSpec)
	if err != nil {
		return fmt.Errorf("registry/teams: %w", err)
	}
	for _, rec := range teamRecords {
		var spec teams.TeamSpec
		if err := json.Unmarshal(rec.Data, &spec); err != nil {
			return fmt.Errorf("registry/teams: %s: %w", rec.ID, err)
		}
		if err := teamsReg.Add(spec); err != nil {
			return err
		}
	}

	routineRecords, err := st.ListRecords(kindRoutine)
	if err != nil {
		return fmt.Errorf("registry/routines: %w", err)
	}
	for _, rec := range routineRecords {
		var spec routines.RoutineSpec
		if err := json.Unmarshal(rec.Data, &spec); err != nil {
			return fmt.Errorf("registry/routines: %s: %w", rec.ID, err)
		}
		if err := routinesReg.Add(spec); err != nil {
			return err
		}
	}

	policyRecords, err := st.ListRecords(kindPolicySpec)
	if err != nil {
		return fmt.Errorf("registry/policies: %w", err)
	}
	for _, rec := range policyRecords {
		var spec policies.PolicySpec
		if err := json.Unmarshal(rec.Data, &spec); err != nil {
			return fmt.Errorf("registry/policies: %s: %w", rec.ID, err)
		}
		if err := policiesReg.Add(spec); err != nil {
			return err
		}
	}
	return nil
}
