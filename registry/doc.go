// Package registry is aa's durable definition store.
//
// It holds reusable specifications — roles, models, teams, capabilities,
// routines, and policies — never runtime instances such as workers, crews,
// assignments, executions, projects, or workflows. Durable definitions live
// here; runtime instances live with the component that runs them.
//
// `roles`, `capabilities`, `models`, `teams`, `routines`, and `policies` are
// implemented definition stores backed by contracts v2. The control plane serves
// them and `app` loads persisted definitions at startup. See
// docs/modules/registry and docs/current-state.md.
package registry
