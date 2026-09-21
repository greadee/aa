# Documentation System

Documentation is part of implementation, not a cleanup activity.

It should preserve:

- phase intent;
- starting state;
- required end state;
- issues/features;
- implementation knowledge;
- architectural decisions;
- pitfalls/solutions;
- sprint execution history.

## Folder Philosophy

Use a linked folder hierarchy rather than one enormous Markdown file.

Recommended:

```text
docs/
├── index/
├── phases/
├── issues/
├── features/
├── architecture/
└── reference/
```

Every meaningful document should be reachable from an index.

## Recommended Tree

```text
docs/
│
├── index/
│   ├── README.md
│   ├── phases.md
│   ├── features.md
│   ├── architecture.md
│   └── sprints.md
│
├── phases/
│   └── ph{N}-{scope}/
│       ├── plan.md
│       ├── plan.uml
│       ├── summary.md
│       ├── summary.uml
│       ├── adr.md
│       └── issues/
│           ├── issue-123-name.md
│           └── ...
│
├── issues/
│   ├── README.md
│   ├── <problem-slug>.md          umbrella problem document
│   └── <branch>-<slug>.md         sub-problem document
│
├── features/
│   └── feature-name/
│       ├── README.md
│       ├── implementation.md
│       ├── testing.md
│       └── pitfalls.md
│
├── architecture/
│   ├── README.md
│   ├── system-overview.md
│   ├── data-flow.md
│   └── diagrams/
│
└── reference/
    ├── cli.md
    ├── schemas.md
    ├── configuration.md
    └── terminology.md
```

## Index Files

Indexes should primarily:

- link;
- summarize;
- orient.

They should not duplicate all downstream content.

## Phase Folder

Required:

```text
plan.md
plan.uml
summary.md
summary.uml
adr.md
issues/
```

The plan documents (`plan.md`, `plan.uml`) are authored at the start of the phase; the summary documents (`summary.md`, `summary.uml`) are authored at the end. The branch name equals the folder name. See [phase-documentation.md](phase-documentation.md).

Optional:

```text
testing.md
migration.md
architecture.md
```

## Phase README

The phase README is the navigation hub.

Recommended content:

```markdown
# Phase N — Name

## Objective

## Status

## Plan
[Phase Plan](./plan.md) · [Plan UML](./plan.uml)

## Summary
[Phase Summary](./summary.md) · [Summary UML](./summary.uml)

## Issues
- [#123 ...](./issues/issue-123-name.md)

## Architecture Decisions
[Phase ADRs](./adr.md)

## GitHub
- Milestone: ...
- Tracking Issue: #...
- Phase PR: #...
- Branch: `ph{N}-{scope}`
```

## Phase Scope

Scope is recorded inside `plan.md`. It should answer:

1. What exists when the phase begins?
2. What is in scope?
3. What must exist when it ends?
4. What is explicitly out of scope?

Recommended section in `plan.md`:

```markdown
## Starting State

## Objective

## In Scope
- ...

## Out of Scope
- ...

## Required End State
- [ ] ...

## Issues
- [#123 ...](./issues/issue-123-name.md)
```

## Starting-State Snapshot

Record relevant existing system behavior.

Example:

```text
Available:
- transaction storage
- manual importer
- CLI portfolio listing

Missing:
- brokerage integration
- account synchronization

Known Constraints:
- DuckDB persistence remains
- ingestion must stay idempotent
```

Where useful, record the source ref:

```text
Phase starting ref:
main @ abc1234
```

## Required End State

Define outcomes, not vague activities.

Weak:

```text
work on broker integration
```

Strong:

```text
broker accounts can be authenticated, discovered, synchronized, normalized, and reconciled without duplicate transactions
```

## Issue Markdown Files

Each meaningful phase issue should have a durable Markdown file:

```text
docs/phases/<phase>/issues/issue-<github-number>-<short-name>.md
```

Recommended content:

```markdown
# #123 — Title

## Type

## Status

## Parent Phase

## GitHub
Issue: #123
PRs:
- #140

## Goal

## Starting State

## Requirements

## Acceptance Criteria
- [ ] ...

## Implementation Notes

## Architecture Decisions
- [ADR-P4-003 — ...](../adr.md#...)

## Dependencies

## Pitfalls and Solutions

## Validation
```

GitHub tracks execution state.

The issue Markdown file preserves durable technical knowledge.

## Umbrella And Sub-problem Documents

A problem that spans more than one module, phase, or branch is recorded as an **umbrella document** under `docs/issues/` and decomposed into sub-problems. Each sub-problem has its own document under `docs/issues/`, authored on the branch that owns it, and links that branch's module update plan and summary. See [issues-and-subproblems.md](issues-and-subproblems.md).

## Module Update Documentation

Work that belongs to a module whose phase already merged is delivered as a module update on the retained phase branch, with the plan and summary beside the module:

```text
<module>/module-upd-plan.md
<module>/module-upd-summary.md
```

For a documentation-only update, the plan and summary sit with the documentation they change. Together they are the durable record of the update, exactly as `plan.md` and `summary.md` are for a phase. See [module-updates.md](module-updates.md).

## Feature Folders

Create a feature folder only when a subsystem is substantial or spans multiple issues/phases.

Example:

```text
docs/features/broker-sync/
├── README.md
├── implementation.md
├── testing.md
└── pitfalls.md
```

Link back to phases and related issues.

## Documentation Updates

Update docs when:

- phase scope changes;
- issue implementation materially diverges from plan;
- an ADR is accepted;
- an important pitfall appears;
- API/schema behavior changes;
- a feature stabilizes;
- work is deferred;
- a phase completes.

Do not wait until the final commit to reconstruct documentation from memory.

## Phase Documentation Completion

Before phase merge:

```text
[ ] plan.md and plan.uml authored at phase start
[ ] summary.md and summary.uml complete
[ ] starting state documented
[ ] required end state verified
[ ] significant issues have issue docs
[ ] issue docs link relevant ADRs
[ ] ADR file reflects accepted decisions
[ ] decisions affirmed and deviations recorded
[ ] important pitfalls recorded
[ ] deferred work identified
[ ] indexes link the phase
[ ] feature docs updated
[ ] architecture/reference docs current
```

## Durable Knowledge Rule

Any knowledge that would be expensive to rediscover should graduate from agent context into repository documentation.
