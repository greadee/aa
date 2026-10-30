# Definitions of Done

Status: **final** — finalized in Sprint 3 (2026-10-30). Revisit only when the
repository changes its release model or GitHub workflow.

These definitions are the minimum bar. A phase or audit may add stricter gates;
it may not waive these without recording the waiver on the issue and PR.

## Issue Definition of Done

An issue is done when all of the following are true.

- [ ] The goal and user-visible or operational value are stated.
- [ ] Requirements describe behavior, not just implementation steps.
- [ ] Acceptance criteria are observable, testable, and all checked.
- [ ] Dependencies, blockers, severity, labels, and milestone are correct.
- [ ] The implementation or documentation change is merged to the owning branch.
- [ ] Tests or validation exist for the change, including failure paths where relevant.
- [ ] Documentation is updated when behavior, contracts, workflow, or state changes.
- [ ] The closeout comment links the commit/PR and records the evidence.
- [ ] Deferred work has a linked follow-up issue; it is not silently dropped.

Documentation-only issues satisfy the same bar: the rendered result, links,
anchors, and reproducibility are the validation.

## Pull Request Definition of Done

A pull request is ready for review when all of the following are true.

- [ ] The PR states its purpose and links every issue it closes or advances.
- [ ] The diff is coherent, scoped, and contains no unrelated drive-by changes.
- [ ] Commit history is readable and follows the repository commit style.
- [ ] Validation commands and results are recorded; required checks are green.
- [ ] Failure paths, boundaries, and migration/rollback impact are considered.
- [ ] Documentation and contract changes are included where required.
- [ ] No secret, credential, generated secret, or local state is committed.
- [ ] For phase, audit, or security work: no unresolved P0/P1 finding; a signed
      audit or explicit residual-risk note is included.
- [ ] The top comment dashboard is current when the PR carries multiple issues.
- [ ] The PR is marked ready only when known work is described and tested.

## Closeout Evidence

Every closed issue gets a short evidence comment:

```text
Done in <commit/PR>.
Validation: <commands and result>.
Residual risk / follow-up: <none or linked issue>.
```

## Waivers

A waiver is valid only when it is written on the PR and linked from the issue.
The PR must name the owner, the reason, the risk, and the follow-up issue or
phase that removes the waiver.
