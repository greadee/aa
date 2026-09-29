// Package registry is aa's durable definition store.
//
// It holds reusable specifications — roles, models, teams, capabilities,
// routines, and policies — never runtime instances such as workers, crews,
// assignments, executions, projects, or workflows. Durable definitions live
// here; runtime instances live with the component that runs them.
//
// The subpackages are introduced structurally by the architecture refactor:
// `roles` and `capabilities` carry the existing vocabularies, while `models`,
// `teams`, `routines`, and `policies` are reserved for their specification
// types (contracts v2). See docs/modules/registry and
// docs/updates/architecture-refactor-1.
package registry
