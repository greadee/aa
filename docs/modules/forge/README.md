# aa-forge — module documentation

> Project history for the `aa-forge` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-forge.md`](../../../forge/aa-forge.md) · **Source** — [`forge/`](../../../forge/) · **Updates** — [`updates/`](./updates/)

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

## Must not

- Merge to `main` without a human gate.
- Store secrets or credentials.
- Be required for local-only work (the fake/local git path must work offline).

## Interfaces

- Forge API (repos, issues, PRs, milestones, releases).
- Kernel RPC for checkpoint and lifecycle events.
- Memory sync for issue and checkpoint records.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`fake`](./fake.md) | Package fake is a deterministic, in-memory forge implementation. |
| [`lifecycle`](./lifecycle.md) | Package lifecycle drives a complete phase lifecycle over a Forge. |
| [`memsync`](./memsync.md) | Package memsync maps forge objects to aa-contracts records and syncs them through a Sink. |
| [`rpc`](./rpc.md) | Package rpc exposes the forge over the aa inter-module RPC v1 methods. |
| [`template`](./template.md) | Package template renders aa's versioned process templates. |

