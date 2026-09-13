# aa-forge.md

Agent directive for the `forge` module (aa-forge).

## Responsibility

Automate the Git and GitHub engineering lifecycle as first-class, auditable objects, and keep the durable record in `memory` in sync.

## Owns

- The forge abstraction with GitHub, local git, and fake implementations.
- Repository and project creation.
- Milestones, phase tracking issues, and checkpoints.
- Issues, branches, child PRs, and the long-lived Draft phase PR.
- Audits and their findings.
- Releases and tags.
- Template rendering from `docs/github-os/templates`.
- Forge-to-memory synchronization.

## Must Not

- Merge to `main` without a human gate.
- Store secrets or credentials.
- Be required for local-only work (the fake/local git path must work offline).

## Interfaces

- Forge API (repos, issues, PRs, milestones, releases).
- Kernel RPC for checkpoint and lifecycle events.
- Memory sync for issue and checkpoint records.

## Rules

1. Every forge operation is idempotent and auditable.
2. The phase branch name equals the phase folder name (`ph{N}-{scope}`).
3. The Draft phase PR exists from phase start to merge.
4. Templates are data, versioned with the process docs.
5. The fake forge makes the entire lifecycle testable without network.

## Canonical references

- [GitHub OS](../docs/github-os/README.md)
- [Phase documentation](../docs/github-os/phase-documentation.md)
