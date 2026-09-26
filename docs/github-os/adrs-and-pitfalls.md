# ADRs and Pitfalls

## ADR Organization

Architectural Decision Records live in the canonical store:

```text
docs/adr/ADR-NNNN-<slug>.md
```

- **Global, chronological ids** (`ADR-NNNN`), one decision per file.
- The **[ADR index](../adr/README.md)** maps the historical phase-scoped ids
  (`ADR-P{phase}-{number}`, `ADR-P{phase}U-{number}`) to their global ids.
- New decisions **append**; ids are never reused or renumbered.
- Historical decisions were backfilled in
  [architecture-refactor-1](../updates/architecture-refactor-1/plan.md) (slice 2).

Historical ids (legacy, preserved for reference):

```text
ADR-P2-001
ADR-P4-003
ADR-P4U-008
```

## ADR Format

```markdown
# ADR-NNNN — Broker credentials remain provider-owned

| Field | Value |
|---|---|
| Status | Accepted |
| Legacy id | `ADR-P4-003` (when backfilled) |
| Source | link to the originating plan/summary |

## Context

## Decision

## Alternatives Considered

## Consequences

## Related Issues

- [#123 ...](../issues/issue-123-name.md)
```

Statuses:

```text
Proposed
Accepted
Superseded
Deprecated
Rejected
```

## Linking Issues and ADRs

Every issue affected by an ADR should link directly to the ADR anchor.

Example inside issue doc:

```markdown
## Architecture Decisions

- [ADR-NNNN — Broker credentials remain provider-owned](../adr/ADR-NNNN-broker-credentials-remain-provider-owned.md)
```

The ADR should link back to affected issues.

Do not duplicate the ADR's full reasoning into every issue document.

## Supersession

Do not delete old ADRs when architecture changes.

Mark them `Superseded` and link the replacement.

The point is to preserve decision history.

## Pitfalls and Solutions

Record implementation knowledge that is expensive to rediscover.

A pitfall is worth documenting when it:

- caused a bug;
- consumed meaningful debugging time;
- involves unintuitive library behavior;
- reflects a domain-specific constraint;
- creates a likely future regression;
- required a workaround;
- affects testing;
- affects deployment/configuration;
- affects future feature work.

## Pitfall Format

```markdown
## Pitfalls and Solutions

### Pitfall: Duplicate transactions after retry

**Observed**

Describe the failure.

**Cause**

Describe root cause.

**Solution**

Describe the applied solution.

**Regression Protection**

Name the test/check.

**Future Guidance**

Explain what future implementations must remember.
```

## Central Feature Pitfall Files

For large subsystems, aggregate recurring lessons in:

```text
docs/features/<feature>/pitfalls.md
```

Individual issue docs may contain detailed discoveries while the feature-level pitfall file links/condenses the most reusable lessons.

Avoid unnecessary duplication.
