# aa-strgy.md

Aspect directive for the **strategy repository** inside the `memory` module.

## Responsibility

Own durable, validated strategies: reusable approaches, decision patterns, pitfalls-with-solutions, and playbooks. Strategies are promoted from learning candidates produced by the kernel and are never invented by the memory module.

## Owns

- Canonical strategy records with provenance and applicability.
- Lifecycle and versioning of strategies (`CANDIDATE → VALIDATED → ACTIVE → SUPERSEDED → ARCHIVED`).
- Links from strategies to projects, roles/trades, issues, and work packages that produced or used them.
- Retrieval for the context compiler.

## Must Not

- Generate strategies (the kernel's job-learning engine proposes candidates).
- Promote unvalidated candidates.
- Store raw session transcripts.

## Rules

1. A strategy requires evidence before validation and activation.
2. Strategies are provider- and machine-independent.
3. Supersede, never overwrite; keep the trail.
4. The context compiler may cite a strategy only if it is `ACTIVE`.

## Canonical references

- [Job learning](../kernel/aa-joblearn.md)
- [Terminology — memory lifecycle](../docs/reference/terminology.md)
