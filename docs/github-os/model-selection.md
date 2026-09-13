# Model Selection

Model assignment should be part of sprint planning.

Use the least expensive model class that can reliably complete the slice.

Do not bind this document to permanent product model names. Map the current available models to the capability tiers below.

## Tier 1 — Lightweight

Use for deterministic or low-ambiguity work.

Suitable:

- file moves/renames;
- mechanical API updates;
- repetitive tests following an existing pattern;
- documentation link updates;
- formatting;
- lint fixes;
- applying an already-defined schema;
- implementing a very clear small function;
- repetitive boilerplate;
- running checks and summarizing failures.

Examples:

```text
add missing type hints
update docs index links
rename provider class throughout package
add unit tests following existing pattern
fix lint violations
```

## Tier 2 — General Coding

Use for ordinary feature implementation where:

- requirements are clear;
- architecture already exists;
- several files may change;
- moderate debugging is expected;
- tests must be designed;
- implementation uses familiar patterns.

Suitable:

```text
implement CRUD repository
add ingestion endpoint
build CLI command
implement provider adapter
add service-level tests
extend scheduler
```

This should handle most sprint slices.

## Tier 3 — Strong Reasoning

Reserve for:

- architecture;
- ambiguous system behavior;
- difficult debugging;
- concurrency;
- migrations;
- complicated Git reconstruction;
- security-sensitive design;
- non-local refactoring;
- dependency restructuring;
- unclear test failures;
- architectural review;
- final phase audit;
- cross-subsystem reasoning.

Examples:

```text
design ingestion-provider boundaries
resolve conflicting normalization semantics
determine safe migration path
analyze synchronization race condition
audit phase architecture
reconstruct branch ancestry
```

## Escalation

Escalate when the current model cannot confidently resolve:

- architecture;
- conflicting requirements;
- repeated test failures;
- non-local side effects;
- ambiguous ownership;
- unexpected regressions;
- unsafe migrations.

Do not repeatedly brute-force an architectural problem with an underpowered model.

## De-Escalation

Once a strong model reduces a problem to deterministic implementation, move execution back to cheaper tiers where appropriate.

Preferred pattern:

```text
Strong reasoning
      ↓
architecture / constraints
      ↓
General coding
      ↓
bounded implementation
      ↓
Lightweight
      ↓
mechanical tests / docs / cleanup
      ↓
Strong reasoning
      ↓
review / audit
```

## Example Assignment

| Slice | Work | Model Class | Expected Commits |
|---|---|---|---:|
| 1 | inspect transaction pipeline | Strong | 0–1 |
| 2 | define normalization contract | Strong | 1 |
| 3 | implement models | General | 1–2 |
| 4 | implement provider adapter | General | 2 |
| 5 | repository integration | General | 1–2 |
| 6 | unit tests | Lightweight / General | 1–2 |
| 7 | idempotency protection | Strong → General | 1–2 |
| 8 | retry regression tests | General | 1 |
| 9 | docs + ADR links | Lightweight | 1 |
| 10 | sprint review/audit | Strong | 0–1 |
