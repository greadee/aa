# aa-forge

Git and GitHub orchestration: projects, issues, pull requests, checkpoints, audits, and releases.

- **Owns:** forge abstraction (GitHub, local git, fake); repo/project creation; milestones and phase tracking; issues; branches; child and phase PRs; checkpoints; audits; releases/tags; template rendering; forge↔memory sync.
- **Must not:** merge without a human gate or store secrets.
- **Status:** in progress (`ph5-forge`).
- **Directive:** [aa-forge.md](aa-forge.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)

## Packages

| Package | Purpose |
|---|---|
| `forge` (module root) | Domain types, capability interfaces, validation rules, idempotency keys, audit log, clock |
| `forge/fake` | Deterministic in-memory implementation used for tests and the e2e lifecycle |
| `forge/template` | Renders the versioned process templates from `docs/github-os/templates` |
| `forge/memsync` | Deterministic mapping of forge objects to `contracts` records |
| `forge/lifecycle` | Phase lifecycle orchestration with the human merge gate |
| `forge/rpc` | Transport-agnostic handlers for the `forge.*` RPC methods |

