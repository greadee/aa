# aa-joblearn.md

Aspect directive for **job learning** inside the `kernel` module.

## Responsibility

Turn completed work into evidence, and evidence into learning candidates. The kernel attributes and scores outcomes; `memory` validates and promotes knowledge. Learning never rewrites history and never silently changes behavior.

## Owns

- Outcome attribution: linking results, tests, gates, and telemetry to roles, trades, workers, and work packages.
- Scoring: success/failure, cost, duration, retries, and quality signals.
- Candidate generation: patterns, strategies, pitfalls, and routing/selection hints, each with provenance.
- Comparative evaluation: backtests of learned suggestions against deterministic baselines.

## Must Not

- Promote knowledge directly (candidates go to `memory`).
- Make a learned signal authoritative for routing, approval, or execution.
- Train or fine-tune models as a prerequisite for learning.

## Evidence gates

Each learned capability (similarity, retrospective, estimator, recommender, conflict engine, learned routing) is disabled until:

1. sufficient attributed outcomes exist;
2. a backtest shows improvement over the deterministic baseline;
3. the capability never bypasses a human gate or an execution contract;
4. a deterministic fallback remains available.

## Interfaces

- Consumes: `memory` history, `obsv` streams, telemetry, gate outcomes.
- Produces: learning candidates (`MEMORY_CANDIDATE_CREATED`) and evaluation reports.

## Canonical references

- [Architecture — learning and optimization](../docs/architecture/README.md#11-end-to-end-phase-plan)
- [Memory promotion](../docs/reference/terminology.md#3-memory-hierarchy)
