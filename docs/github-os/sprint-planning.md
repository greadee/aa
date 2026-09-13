# Sprint Planning

## Philosophy

A phase should normally be executed through one or more planned sprints.

A sprint is a coherent implementation progression toward the phase's required end state.

Each sprint is broken into **slices**.

A slice is the smallest planned unit that:

- produces a coherent technical outcome;
- can be given to an agent with bounded context;
- can normally result in one or several meaningful commits;
- can be validated independently;
- advances one or more issues.

## Typical Size

A typical sprint should contain approximately:

```text
6–12 slices
```

and may reasonably produce:

```text
10–20 commits
```

across the phase branch and child issue branches.

These are planning ranges, not quotas.

Do not manufacture commits merely to reach a target.

## Example Sprint

```text
Sprint: Broker Foundation

Slice 1 — inspect existing ingestion architecture
Slice 2 — define provider contract
Slice 3 — implement authentication model
Slice 4 — implement account discovery
Slice 5 — add normalized broker account models
Slice 6 — persistence integration
Slice 7 — unit tests
Slice 8 — integration tests
Slice 9 — failure/retry handling
Slice 10 — documentation and ADR updates
```

Possible commits:

```text
add broker provider interface
add broker authentication models
add broker account discovery
add broker account persistence
fix provider error normalization
add broker provider unit tests
add fake broker provider for integration tests
add account synchronization integration tests
fix retry state handling
add retry regression tests
docs: add broker provider ADR
update phase sprint progress
```

## Sprint Plan File

Each phase maintains its plan at:

```text
docs/phases/ph{N}-{scope}/plan.md
```

The phase plan **is** the sprint plan. A slice maps to exactly one commit. See [phase-documentation.md](phase-documentation.md).

Recommended structure:

```markdown
# Sprint Plan

## Phase

## Sprint 1 — Name

### Objective

### Issues
- #123
- #124

### Slices

#### Slice 1 — Name

Model Class: ...

Goal:
...

Commit Message:
...

Validation:
...

## Sprint Exit Criteria
- [ ] ...
```

## Slice Template

Each slice should specify:

```markdown
### Slice N — <name>

**Issues**

#123, #127

**Goal**

**Inputs**

**Expected Output**

**Model Class**

**Commit Message**

**Validation**

**Dependencies**

**Documentation**
```

## Slice Boundaries

Good:

```text
implement normalized broker transaction model
```

Too broad:

```text
build broker sync
```

Too narrow:

```text
add one import statement
```

A slice should fit one focused agent task with enough context to understand and validate the result.

## Slices and Issues

One issue may contain several slices.

Example:

```text
Issue #125 — Transaction Synchronization

Slice 1 — normalization design
Slice 2 — transaction models
Slice 3 — provider adapter
Slice 4 — persistence
Slice 5 — tests
Slice 6 — retry handling
```

One slice may occasionally advance multiple tightly related issues.

Document relationships explicitly.

## Slices and Commits

A slice maps to **exactly one commit**. This planning invariant keeps the phase plan directly traceable to Git history.

```text
slice  →  commit
```

The slice's **Commit Message** field in `plan.md` is the exact message that will be used. If a slice turns out to need more than one commit, split the slice. If it needs none, fold it into a neighboring slice.

Example:

```text
Slice: provider adapter

commit:
add broker transaction provider adapter
```

## Slices and PRs

Several slices may belong to one child PR.

Issue-level PRs should remain reviewable units.

A sprint may therefore contain several child PRs.

## Progress States

Use:

```text
Planned
Ready
In Progress
Blocked
Review
Complete
Deferred
```

Example:

```markdown
| Slice | Status | Issue | PR | Model |
|---|---|---|---|---|
| Provider contract | Complete | #127 | #140 | Strong |
| Account discovery | Complete | #124 | #143 | General |
| Transaction normalization | In Progress | #125 | #148 | Strong |
| Retry tests | Planned | #131 | — | General |
```

## Sprint Progress Comments

At meaningful checkpoints, update the phase PR.

Example:

```markdown
## Sprint 2 Update — Transaction Pipeline

Completed slices:
- normalization contract
- transaction models
- provider adapter
- persistence integration

In progress:
- idempotency
- retry tests

Commits added this sprint:
11

Child PRs:
- #148 ...
- #153 ...

Current blocker:
...

Next:
...
```

## Replanning

Replan when:

- architecture changes;
- a blocking bug appears;
- a feature is larger than expected;
- an issue must split;
- a dependency changes;
- audit findings change priorities.

Record substantial replanning.

Example:

```markdown
## Sprint Replan — YYYY-MM-DD

### Reason

### Changes
- ...

### Impact
No change to phase exit criteria.
```

## Sprint Retrospective

At sprint end, record:

- planned slices;
- completed slices;
- deferred slices;
- issues completed;
- PRs merged;
- commits produced;
- ADRs created;
- pitfalls discovered;
- tests added;
- blockers remaining.

Recommended:

```markdown
## Sprint Retrospective

### Planned
10 slices

### Completed
9 slices

### Deferred
1 slice -> Sprint 3

### Repository Changes
14 commits
3 child PRs merged
4 issues completed

### Decisions
- ADR-P4-005

### Pitfalls Learned
- ...

### Follow-Up
- #166 ...
```
