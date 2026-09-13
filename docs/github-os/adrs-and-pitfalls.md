# ADRs and Pitfalls

## ADR Organization

Architectural Decision Records should be grouped by phase in:

```text
docs/phases/<phase>/adr.md
```

Use stable identifiers:

```text
ADR-P{phase}-{number}
```

Examples:

```text
ADR-P2-001
ADR-P4-003
```

## ADR Format

```markdown
## ADR-P4-003 — Broker credentials remain provider-owned

### Status

Accepted

### Context

### Decision

### Alternatives Considered

#### Alternative A

#### Alternative B

### Consequences

### Related Issues

- [#123 ...](./issues/issue-123-name.md)
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

- [ADR-P4-003 — Broker credentials remain provider-owned](../adr.md#adr-p4-003--broker-credentials-remain-provider-owned)
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
