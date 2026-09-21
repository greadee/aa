# Phase documentation

This folder contains one directory per phase, named `ph{N}-{scope}` (for example `ph0-scaffold`). The **branch name is identical to the folder name**.

## Required artifacts per phase

```
docs/phases/ph{N}-{scope}/
├── plan.md       phase plan, broken into slices (each slice = one commit)
├── plan.uml      planned infrastructure diagrams (UML)
├── summary.md    phase summary: delivered, decisions affirmed, deviations
└── summary.uml   plan diagrams copied and edited to the as-built state
```

### plan.md

Authored at the **start** of the phase. Contains: objective, starting state (repo `main @ <sha>`, available/missing, constraints), in scope, out of scope, the slice breakdown, exit criteria, and the test plan. The slice breakdown is the sprint plan and each slice names the commit message it will produce.

### plan.uml

Authored with `plan.md`. Uses PlantUML (`.uml`) to show the intended infrastructure: component/context, deployment, class, and sequence diagrams describing what will exist when the phase completes.

### summary.md

Authored at the **end** of the phase. Records: delivered slices and commits, the PR(s), tests and gates, architectural decisions that proved correct, deviations from the plan and the reason for each, follow-ups, and an audit result.

### summary.uml

The `plan.uml` copied and edited so it reflects the finished system. Together, `summary.md` + `summary.uml` are the phase summary of record.

## Phase parts

A large phase may be split into **parts** (a., b., c., …) when it spans genuinely different workstreams with different dependencies or review needs. A split phase keeps its single `plan.md` (the overview and part index) and adds one plan per part:

```
docs/phases/ph{N}-{scope}/
├── plan.md
├── plan.uml
├── summary.md
├── summary.uml
└── parts/
    ├── a-{part-scope}/
    │   ├── plan.md
    │   ├── plan.uml
    │   ├── summary.md
    │   └── summary.uml
    └── b-{part-scope}/
        └── ...
```

Rules:

- A part never creates a phase or renames the branch; all parts share the phase branch and the phase PR.
- Each part authors its own `plan.md` and `plan.uml` at its start, and its own `summary.md` and `summary.uml` at its end, to the same standard as a phase.
- Part slices map to commits on the phase branch (or on a child branch merged into it). A part is complete when its own required end state and exit criteria pass.
- The phase is complete when its planned parts are complete and the phase PR merges to `main`.

## Index

Tracked in [../index/phases.md](../index/phases.md).
