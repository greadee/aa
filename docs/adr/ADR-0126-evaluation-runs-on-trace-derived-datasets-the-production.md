# ADR-0126 — Evaluation runs on trace-derived datasets; the production baseline is the `sifter` recommender behind the `Predictor` seam

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P9U-005` |
| **Source** | [docs/modules/kernel/updates/joblearn-trace-distillation/plan.md](../modules/kernel/updates/joblearn-trace-distillation/plan.md) |
| **Scope** | module update · kernel/joblearn |

## Context

Backfilled from the original decision record for **module update 9 (kernel/joblearn)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Evaluation runs on trace-derived datasets; the production baseline is the `sifter` recommender behind the `Predictor` seam

## Rationale

Closes phase 9 deviation 3 without importing `sifter` or calling a model in the engine

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/kernel/updates/joblearn-trace-distillation/plan.md).

## Related

- Legacy id `ADR-P9U-005`
- [ADR index](./README.md)
