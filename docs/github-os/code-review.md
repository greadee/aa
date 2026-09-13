# Code Review

Every non-trivial implementation PR should receive review before completion.

## Review Scope

Inspect:

1. correctness;
2. scope;
3. architecture;
4. maintainability;
5. test quality;
6. error handling;
7. security implications;
8. backwards compatibility;
9. unnecessary complexity;
10. documentation where relevant.

Review the diff in system context, not only line-by-line.

## Agent Self-Review

Before requesting external review:

- inspect the complete diff;
- confirm it matches the linked issue;
- look for accidental unrelated changes;
- check error paths;
- inspect new public interfaces;
- confirm tests validate meaningful behavior;
- run relevant validation;
- remove debugging artifacts;
- synchronize docs where needed.

Passing tests alone is not proof of correctness.

## Review Output

For substantial PRs:

```markdown
## Review Summary

### Blocking
- ...

### Non-Blocking
- ...

### Validation
- tests reviewed
- tests executed
- lint/type status

### Risk

Low / Medium / High

### Recommendation

Approve / Request Changes / Needs Investigation
```

Do not manufacture findings merely to look thorough.

## Severity

```text
P0 — critical
P1 — high
P2 — medium
P3 — low
```

Interpretation:

- **P0:** data loss, security compromise, severe correctness failure, repository-breaking defect.
- **P1:** major feature failure, serious architecture problem, likely production failure.
- **P2:** meaningful maintainability/correctness edge case/test weakness/design concern.
- **P3:** minor cleanup/readability/naming/optional improvement.

P0/P1 normally block merge.

P2 may block depending on context.

P3 generally should not block unless collectively serious.

## Findings Outside Current Scope

If review finds a legitimate issue outside scope:

1. decide whether it must block;
2. create a GitHub issue if it should survive the session;
3. link it in the review;
4. assign it to current/future phase;
5. continue if non-blocking.

Do not force unrelated cleanup into the current PR.
