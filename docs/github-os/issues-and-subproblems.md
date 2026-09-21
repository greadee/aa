# Issues, Problems, and Sub-problems

A **problem** is a deficiency or a goal. Work that must remain visible beyond the current session becomes an **issue**. A problem that spans more than one module, phase, or branch is recorded as an **umbrella problem** and decomposed into **sub-problems**, each owned and solved by exactly one branch.

This module extends [issues-and-stories.md](issues-and-stories.md). That module defines how to write a single issue; this one defines how a problem too large for one branch is split into several, how each sub-problem is owned, and where its durable documentation lives.

## Problem, Sub-problem, Issue, Document

| Term | Meaning |
|---|---|
| Problem (umbrella) | A deficiency or goal too large or too cross-cutting for one branch |
| Sub-problem | One distinct, independently solvable part of a problem |
| Issue | The GitHub execution record for a sub-problem |
| Issue document | The durable knowledge record under `docs/issues/` |

Distinct problems are separate issues. Do not collapse several problems into one document, and do not solve a sub-problem on a branch that does not own it.

## Decomposition

A problem statement answers what is wrong, who it affects, and why it matters. It is recorded as an **umbrella document** under `docs/issues/`, one file per umbrella change:

```text
docs/issues/<problem-slug>.md
```

Recommended structure:

```text
# ISS-<AREA>-<slug> — <problem>

Type:
Status: open (umbrella)
Affects branches:

## Problem

## Scope boundary

## Sub-problems

| Id | Problem | Module / branch | Why it matters | Status | Document |

## Dependency order

## Ownership
```

The umbrella:

- states the problem and the distinct sub-problems;
- records the dependency order between sub-problems;
- links each sub-problem's document;
- explicitly does **not** specify the solution for a branch whose work has not started.

Each sub-problem's solution plan is authored with the pull request on the branch that owns it, and is linked from that sub-problem's document. The umbrella states the problem and the order; the branch states the solution.

## Ownership

Every sub-problem belongs to one module, and each module is delivered by its phase branch (`ph{N}-{scope}`). A sub-problem is therefore solved on the phase branch that owns its module.

- If the owning phase has not merged, the work continues on its branch.
- If the owning phase has already merged, **reopen its retained branch** for a module update. Do not create a new phase for work that belongs to an existing one. See [module-updates.md](module-updates.md).
- The branch name never changes. The retained branch carries the new commits and a new pull request into `main`.

A sub-problem's document names the branch that owns it and never prescribes a solution for a different branch.

## Sub-problem Documents

Each meaningful sub-problem has a durable document under `docs/issues/`, named for its branch and topic:

```text
docs/issues/<branch>-<slug>.md
```

Recommended content:

```text
# ISS-<AREA>-<n> — <module>: <sub-problem>

Type:
Status:
Branch:
Phase PR:
Parent: <link to the umbrella document>
Module plan:
Module summary:

## Goal

## Problem

## Requirements

## Acceptance Criteria
- [ ] ...

## Affected branches

## Notes
```

GitHub tracks execution state. The document preserves durable technical knowledge. The module plan and module summary are authored on the owning branch and linked from this document.

## Dependency Order

Record the order in which sub-problems unblock one another. A sub-problem may be a prerequisite for several others. State which branches it unblocks without specifying their solutions, and keep the order in the umbrella, not in the individual documents.

## Index

Umbrella documents, sub-problem documents, and their branch coverage are indexed in `docs/issues/README.md`. The index links each document to its issue id, scope, and owning branch.

## Agent Discovery Policy

[issues-and-stories.md](issues-and-stories.md#agent-discovery-policy) governs out-of-scope discoveries. When a discovery is another part of a larger deficiency, add it to the umbrella as a sub-problem rather than silently expanding the current branch.

## Related

- Issue types, bodies, and discovery: [issues-and-stories.md](issues-and-stories.md)
- Module update cycle: [module-updates.md](module-updates.md)
- Documentation tree and indexes: [documentation-system.md](documentation-system.md)
- Worked examples: the phase 1 and phase 2 module-update pull requests and their `docs/issues/` records
