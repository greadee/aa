# Phase Management

## Phase Checkpoint Model

Every substantial development phase should be visible as a GitHub checkpoint.

A checkpoint consists of:

1. a GitHub Milestone;
2. a phase tracking issue;
3. a phase branch;
4. a long-lived Draft phase PR into `main`;
5. child issues;
6. child implementation PRs;
7. final review and audit.

Conceptually:

```text
GitHub Milestone
      ↓
Phase Tracking Issue
      ↓
Child Issues
      ↓
Child PRs
      ↓
Phase Branch
      ↓
Draft Phase PR
      ↓
Review + Audit
      ↓
main
```

## Milestone Naming

Use:

```text
Phase {number} — {Human Readable Name}
```

Examples:

```text
Phase 0 — Scaffold
Phase 1 — Contracts
Phase 2 — Memory
Phase 5 — Forge
Phase 10 — Console
```

## Phase Tracking Issue

Preferred title:

```text
Phase {number}: {phase objective}
```

The issue should define:

- objective;
- scope;
- child issues;
- dependencies;
- risks/open questions;
- exit criteria.

Recommended structure:

```markdown
## Objective

## Scope

## Issues
- [ ] #123 ...
- [ ] #124 ...

## Dependencies

## Risks / Open Questions

## Exit Criteria
- [ ] planned functionality implemented
- [ ] relevant tests passing
- [ ] documentation updated
- [ ] no unresolved blocking bugs
- [ ] phase audit complete
- [ ] phase PR reviewed
- [ ] ready to merge
```

When scope changes, update it explicitly.

Do not silently remove abandoned work. Mark it deferred/cancelled and link its destination.

## Phase Branch

The phase branch name is identical to the phase documentation folder name (`ph{N}-{scope}`). See `git-workflow.md`.

## Phase Documentation Artifacts

At phase **start**, create `docs/phases/ph{N}-{scope}/` and author `plan.md` plus `plan.uml`. At phase **end**, author `summary.md` and `summary.uml` by copying and editing the plan diagrams to the as-built state. The branch name must equal the folder name. See [phase-documentation.md](phase-documentation.md).

## Long-Lived Draft Phase PR

Open the phase PR when the phase begins, not after it finishes.

It is the live integration checkpoint and status page.

Target:

```text
phaseN-<name> -> main
```

The phase PR should show:

- phase objective;
- tracking issue;
- milestone;
- feature progress;
- infrastructure progress;
- testing progress;
- docs progress;
- child PRs;
- current status;
- known risks;
- validation;
- exit criteria.

## Phase Definition of Done

A phase is complete only when:

```text
[ ] plan.md and plan.uml authored at phase start
[ ] required phase issues complete
[ ] required child PRs merged
[ ] deferred work has explicit issues
[ ] relevant validation passes
[ ] summary.md and summary.uml complete
[ ] documentation updated
[ ] phase-end audit complete
[ ] blocking findings resolved
[ ] phase PR progress summary current
[ ] phase PR no longer Draft
[ ] final review complete
[ ] phase branch ready for main
```

## Main Merge

Only the phase PR should normally merge the completed phase branch into `main`.

This produces two useful levels of history:

1. detailed child PRs inside the phase;
2. one coherent milestone merge into `main`.

The phase branch is **retained** after merge and must never be deleted; see [git-workflow.md](git-workflow.md#branch-retention).
