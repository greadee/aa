# Issues and Stories

GitHub issues represent work that should remain visible beyond the current coding session.

## Preferred Types

Conceptual issue types:

```text
feature
user-story
bug
technical-debt
refactor
test
documentation
audit
investigation
```

Use labels if native issue types are unavailable.

## Feature Issues

Preferred title:

```text
Add broker position synchronization
```

Recommended body:

```markdown
## Goal

## Requirements
- ...

## Acceptance Criteria
- [ ] observable behavior
- [ ] tests added/updated
- [ ] error cases handled
- [ ] docs updated where required

## Notes
```

Requirements should emphasize behavior. Avoid prescribing implementation details unless they are real architectural constraints.

## User Stories

Use user-story wording when functionality is best expressed from the user's perspective.

```text
As a <user>,
I want <capability>,
so that <outcome>.
```

Example:

```text
As a portfolio user,
I want broker transactions synchronized automatically,
so that my dashboard reflects brokerage activity without manual import.
```

Do not force technical infrastructure work into user-story form.

## Bug Issues

Preferred title:

```text
Broker import duplicates transactions after partial retry
```

Recommended body:

```markdown
## Problem

## Expected Behavior

## Actual Behavior

## Reproduction
1. ...
2. ...

## Impact

## Evidence

## Suspected Area

## Acceptance Criteria
- [ ] root cause understood
- [ ] bug corrected
- [ ] regression test added where practical
- [ ] related tests pass
```

Keep speculation clearly labeled.

## Technical Debt

Create a technical-debt issue when:

- implementation works but has known architectural weakness;
- temporary duplication is introduced;
- test coverage is deferred;
- an abstraction needs redesign;
- a workaround should later be removed;
- cleanup is too large/unrelated to current PR.

State why it matters and what should trigger addressing it.

## Investigation Issues

Use when implementation should wait for uncertainty to be resolved.

Examples:

```text
Investigate brokerage API pagination behavior
Evaluate DuckDB migration strategy for analytics schema
Determine websocket reconnect requirements
```

Possible outputs:

- design decision;
- ADR;
- one or more implementation issues;
- closure with no implementation.

## When to Create an Issue

Create one when work:

- belongs to a planned phase;
- is more than trivial;
- should survive the current session;
- has dependencies;
- needs acceptance criteria/review;
- represents a bug worth tracking;
- represents deferred work;
- was found during audit;
- should be implemented separately from the current PR.

Do not create an issue for every typo.

## Agent Discovery Policy

When an agent discovers out-of-scope work:

```text
Required for current issue?
    |
    +-- yes -> handle if reasonably related
    |
    +-- no
         |
         +-- meaningful -> create/link issue
         |
         +-- trivial/local -> optionally fix if harmless
```

Do not silently expand scope.

## Feature Discovery

If implementation reveals a useful new feature that is not in scope:

- do not implement automatically;
- create a feature issue;
- explain why it matters;
- note dependencies;
- suggest a future phase.

## Decomposition

A problem too large for one branch, or spanning several modules, is recorded as an umbrella and decomposed into sub-problems, each owned by exactly one branch. See [issues-and-subproblems.md](issues-and-subproblems.md) and [module-updates.md](module-updates.md).
