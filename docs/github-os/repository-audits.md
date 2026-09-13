# Repository Audits

A repository audit is broader than a PR review.

It evaluates codebase health and should primarily identify/classify findings.

Do not automatically fix every finding during audit mode.

## When to Audit

Perform an audit:

- before completing a major phase;
- after a major architectural refactor;
- when explicitly requested;
- when unexplained failures/complexity accumulate;
- before a significant release where appropriate.

## Audit Areas

### Correctness

- suspicious logic;
- unhandled edge cases;
- stale assumptions;
- inconsistent behavior;
- race conditions;
- incorrect transformations.

### Architecture

- dependency direction;
- inappropriate coupling;
- duplicated responsibilities;
- unclear ownership;
- overgrown abstractions;
- phase-boundary violations.

### Tests

- missing critical coverage;
- implementation-coupled tests;
- brittle/skipped tests;
- dead fixtures;
- untested error paths;
- integration gaps.

### Security

- secrets;
- unsafe config;
- input handling;
- authorization;
- dependency concerns;
- sensitive logging.

### Data

- schema inconsistencies;
- migrations;
- type mismatches;
- destructive operations;
- idempotency;
- duplicate handling.

### Reliability

- retries;
- timeouts;
- recovery;
- partial completion;
- concurrency;
- resource cleanup.

### Performance

- obvious N+1 behavior;
- repeated I/O;
- unnecessary allocation;
- avoidable scans;
- blocking operations in critical paths.

### Maintainability

- dead code;
- duplication;
- oversized modules;
- unclear naming;
- circular dependencies;
- stale TODOs;
- unused abstractions.

### Documentation

- stale README content;
- outdated diagrams;
- missing ADRs;
- undocumented environment requirements;
- docs/behavior mismatch.

### Repository Hygiene

- generated files tracked accidentally;
- tracked secrets;
- stale branches;
- inconsistent tooling;
- lockfiles;
- CI correctness;
- unnecessary binary artifacts.

## Audit Finding Requirements

Each meaningful finding should have:

- title;
- classification;
- severity;
- evidence;
- impact;
- recommended action;
- suggested phase;
- affected files/subsystem.

Do not file speculative findings without evidence.

## Audit Issue Format

```markdown
## Finding

## Classification

Bug / Technical Debt / Refactor / Security / Test / Documentation

## Severity

P0 / P1 / P2 / P3

## Evidence

## Impact

## Recommendation

## Scope

## Suggested Phase
```

## Audit Summary

```markdown
## Repository Audit Summary

### Findings

P0: 0
P1: 1
P2: 4
P3: 7

### Blocking
- #...

### Recommended Before Phase Merge
- #...

### Deferred
- #... -> Phase N

### Areas Reviewed
- architecture
- tests
- CI
- docs

### Result

Not ready / Ready with conditions / Ready
```

## Phase-End Audit

Before merging a major phase, verify:

- phase requirements met;
- child issues closed/deferred properly;
- relevant full test suite passes;
- no unresolved high-severity bug;
- architecture coherent;
- docs current;
- phase PR accurately describes delivery;
- no temporary debug code;
- deferred work has issues;
- no important TODO exists only in source comments;
- Git graph is sensible.
