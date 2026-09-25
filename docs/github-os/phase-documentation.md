# Phase Documentation

Every phase produces a durable, self-contained record of what was intended and what was actually built. This is the heart of aa's engineering process: the plan is written before the work, and the summary is derived from it after the work.

## Folder And Branch Naming

Phase folder and branch names are **identical**:

```text
ph{number}-{scope}
```

Examples:

```text
docs/phases/ph0-scaffold/     branch: ph0-scaffold
docs/phases/ph1-contracts/    branch: ph1-contracts
docs/phases/ph5-forge/        branch: ph5-forge
docs/phases/ph9-joblearn/     branch: ph9-joblearn
docs/phases/ph10-console/     branch: ph10-console
```

Rules: no zero-padding, hyphen between number and scope, short lowercase kebab-case scope. The scope names the module or workstream the phase delivers.

## Required Artifacts

Each phase folder contains exactly four required documents, plus optional issue files:

```text
docs/phases/ph{N}-{scope}/
├── plan.md       phase plan and slice breakdown (authored at phase start)
├── plan.uml      planned infrastructure diagrams (authored at phase start)
├── summary.md    phase summary (authored at phase end)
├── summary.uml   as-built diagrams (authored at phase end)
└── issues/       optional durable issue documentation
```

The plan and the summary are **separate files**, and the diagrams are **separate files** from the prose. Diagrams are separate so they can be diffed and evolved independently as the design changes.

## plan.md — At Phase Start

Authored before any implementation. Contains:

1. **Objective** — the outcome, stated so it can be verified.
2. **Starting state** — repo `main @ <sha>`, what exists, what is missing, constraints.
3. **In scope** and **out of scope**.
4. **Architecture decisions** — decisions made for this phase, each with reasoning, linked to `plan.uml`.
5. **Slices** — the sprint plan. Each slice is an independently validatable unit and maps to exactly one commit.
6. **Exit criteria** — the definition of done.
7. **Test plan** — what will be tested and at which layer.

Each slice records:

```markdown
### Slice N — <name>

**Issues** — #123
**Goal**
**Inputs**
**Expected Output**
**Model Class** — Lightweight / General / Strong
**Commit Message** — the exact commit message this slice produces
**Validation**
**Dependencies**
**Documentation**
```

A slice maps to exactly one commit. If a slice needs more than one commit, split it.

## plan.uml — At Phase Start

PlantUML source describing the intended end state of the phase's infrastructure. Depending on the phase, include one or more of:

- component/context diagram — modules, boundaries, and dependencies;
- deployment diagram — daemons, sockets, processes, machines;
- class diagram — key types, interfaces, and ownership;
- sequence diagram(s) — the primary flows the phase enables.

Diagram elements should reference the decisions in `plan.md`.

## summary.md — At Phase End

Authored after the phase is implemented and verified. Contains:

1. **Delivered** — slices and commits as shipped, child PRs, tracking issue.
2. **Validation** — tests, gates, audits, and evidence.
3. **Decisions affirmed** — which planned decisions proved correct, and the evidence.
4. **Deviations** — every deviation from the plan, with the reason it was necessary.
5. **Deferred / follow-up** — work moved out, with destination issues.
6. **Metrics** — commits, PRs, issues, ADRs, pitfalls, test counts.
7. **Retrospective** — what to repeat and what to avoid next phase.

## summary.uml — At Phase End

The `plan.uml` **copied and edited** to reflect the system as actually built. It must show the as-built infrastructure, not the aspirational one. Together with `summary.md`, it is the phase's record of truth.

## Numbered Phases vs Update Phases

- **Numbered phases** (`ph{N}-{scope}`) are reserved for **major additions**.
- **Update phases** (unnumbered, `docs/updates/<slug>/`) cover refactors, edits,
  corrections, and module updates. They carry the same plan/summary discipline with
  lightweight, change-scoped diagrams, and link the module updates they contain
  (`docs/modules/<module>/updates/<slug>/`).
- Architecture decisions live in the global [ADR store](../adr/README.md), not in a
  per-phase `adr.md`.

## Relationship To Other Modules

- Branch and commit conventions: [git-workflow.md](git-workflow.md)
- Milestones, tracking issues, Draft phase PRs, phase DoD: [phase-management.md](phase-management.md)
- Slice philosophy and progress states: [sprint-planning.md](sprint-planning.md)
- ADRs and pitfalls: [adrs-and-pitfalls.md](adrs-and-pitfalls.md)
- Documentation tree and indexes: [documentation-system.md](documentation-system.md)
