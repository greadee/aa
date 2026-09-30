# Sprint index

A phase executes through one or more sprints. A sprint is **6–12 slices** producing roughly **10–20 commits**. A slice is the smallest coherent, bounded-context, independently validatable unit and maps to **exactly one commit**.

## Conventions

- Sprint plan: `docs/phases/ph{N}-{scope}/plan.md` (the phase plan is the sprint plan for single-sprint phases).
- Slice fields: goal, inputs, output, model tier, commit message, validation, dependencies, documentation.
- Progress states: `Planned`, `Ready`, `In Progress`, `Blocked`, `Review`, `Complete`, `Deferred`.
- Model tiers: Lightweight, General, Strong (least-cost capable tier; escalate only when needed).
- Each slice ends with its validation and a commit. Phases stop at slice boundaries for review unless a sprint explicitly chains slices.

## Sprints

| Sprint | Type | Branch | PR | Status |
|---|---|---|---|---|
| [issue-impl-sep28](../updates/issue-impl-sep28/plan.md) | update phase | `dev` | #21 | In Progress |

This sprint carries its own cadence: **one commit per issue/substage**, folding
closely-related sub-tasks, instead of one commit per slice (plan SD-7, scoped to
this sprint only).
