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

## Index

Tracked in [../index/phases.md](../index/phases.md).
