// Package forge is aa's Git and GitHub orchestration module: projects, issues,
// pull requests, checkpoints, audits, and releases.
//
// The package defines the forge abstraction (capability interfaces and domain
// types), the idempotency and audit primitives shared by every implementation,
// and the deterministic rules the phase lifecycle depends on. Implementations
// live in subpackages (for example forge/fake); the kernel reaches a hosted
// forge over RPC.
//
// The forge never merges to main without a human gate, never stores secrets,
// and works offline through a deterministic fake. See docs/architecture/README.md
// and docs/phases/ph5-forge/plan.md.
package forge
