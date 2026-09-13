# Repository Reconstruction

Use this module only when rebuilding, reorganizing, or repairing Git history.

## Objective

Fix repository topology without unnecessarily changing application contents.

Prefer native Git operations over manually reproducing edits.

## Before Changing Anything

Inspect:

```bash
git status
git branch -a
git log --graph --all --decorate --oneline
git remote -v
```

Determine:

- existing branch topology;
- chronological commit order;
- merge commits;
- logical project phases;
- unique branch commits;
- intended branch tips.

Do not rewrite history before understanding the current graph.

## Reconstruction Mapping

Create a plan such as:

```text
source commit(s)         destination
------------------------------------------------
initial setup            phase0-setup
data/domain work         phase1-dataModel
metric ingestion         phase2-metricIngestion
analytics                phase3-analytics
broker integration       phase4-brokerSync
```

Use commit purpose, not only date.

## Replaying History

Prefer Git-native operations:

```bash
git branch
git switch
git cherry-pick
git cherry-pick -x
git rebase
git rebase --onto
git rebase --rebase-merges
git format-patch
git am
git diff
git show
```

Git already knows exact additions, modifications, deletions, and renames.

Do not manually rewrite every historical file edit unless Git cannot reproduce it.

## Preservation Rules

Unless explicitly instructed otherwise:

- preserve actual code changes;
- preserve logical commit order;
- preserve useful commit granularity;
- preserve original author information where possible;
- preserve good historical messages;
- preserve meaningful fixes/refactors as distinct events;
- do not silently omit commits;
- do not combine unrelated work.

Commit SHAs may change after rebase/cherry-pick. That is acceptable when repairing ancestry.

## Rewriting Messages

Do not rewrite a good historical message merely for cosmetic consistency.

Rewrite only when:

- it is meaningless;
- it refers to obsolete branch organization;
- the user requests normalization;
- reconstruction creates a genuinely new logical commit.

## Branch Reconstruction

A later phase should normally branch from `main` after the previous phase is integrated.

Avoid making later phases descend from unrelated unfinished work.

## Conflict Resolution

When a replay conflicts:

1. inspect the original commit;
2. inspect its parent;
3. understand the intended delta;
4. reproduce that delta in the corrected topology;
5. avoid unrelated redesign;
6. run relevant tests;
7. continue.

## Verification

After rebuilding:

```bash
git log --graph --all --decorate --oneline
```

Confirm:

- phase branches start at intended points;
- commits are logically ordered;
- merges happen at intended milestones;
- `main` contains expected completed work.

Compare corresponding old/new snapshots:

```bash
git diff <old-ref> <new-ref>
```

An empty diff is preferred when topology changed but contents should match.

Run tests, lint, and validation.

## Safety

Never destructively rewrite the only copy of a repository.

- preserve the original repository;
- create backup refs where appropriate;
- prefer a new repository or isolated clone;
- do not force-push important remote history without explicit permission;
- validate before replacing remote refs.
