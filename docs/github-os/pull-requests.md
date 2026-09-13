# Pull Requests

## Child Pull Requests

Each meaningful issue branch should normally produce a PR into the active phase branch.

Example:

```text
feature/123-broker-sync
        ↓
phase4-brokerSync
```

PR title should follow repository commit-message tone.

Examples:

```text
add broker position synchronization
fix duplicate transactions after broker retry
```

## Child PR Body

Recommended structure:

```markdown
## Summary

## Related Issue

Closes #123

## Changes
- ...

## Validation
- [ ] unit tests
- [ ] integration tests
- [ ] lint/type checks
- [ ] manual verification if relevant

## Notes

## Phase

Part of #<phase-tracking-issue>
Milestone: Phase N — <name>
```

Use `Closes #123` only if merging truly completes the issue.

Use `Refs #123` when work remains.

## Draft PRs

Use Draft PRs proactively when:

- implementation is underway;
- architecture should be visible early;
- CI feedback is useful;
- several commits will accumulate;
- progress needs visibility.

Move to Ready for Review only when:

- implementation is substantially complete;
- known work is described;
- relevant tests are passing;
- diff is coherent enough to review.

## Phase Pull Request

Open a Draft PR from the phase branch to `main` as soon as the phase begins.

Preferred title:

```text
Phase 4: broker synchronization
```

Historical style such as `merge phase4-brokerSync` is also acceptable, but descriptive phase titles are preferred for new repositories.

## Phase PR Body

Use it as a dashboard:

```markdown
## Phase Objective

## Tracking

Phase issue: #100
Milestone: Phase 4 — Broker Sync

## Progress

### Features
- [x] ...
- [ ] ...

### Infrastructure
- [x] ...
- [ ] ...

### Testing
- [x] ...
- [ ] ...

### Documentation
- [ ] ...

## Child Pull Requests
- [x] #140 ...
- [ ] #148 ...

## Current Status

## Known Issues / Risks

## Validation
- [ ] full test suite
- [ ] lint/type checks
- [ ] architecture docs
- [ ] audit
- [ ] blocking issues resolved

## Exit Criteria
```

## Progress Updates

Post a phase PR comment after meaningful milestones, not after every commit.

Triggers include:

- major child PR merged;
- major feature functional;
- architecture materially changed;
- scope changed;
- blocker discovered/resolved;
- testing milestone;
- final review started;
- audit finished.

Preferred format:

```markdown
## Phase Update — <milestone>

### Completed
- ...

### In Progress
- ...

### Blockers
- ...

### Next
- ...
```

## Scope Change Update

When phase scope changes materially:

```markdown
## Scope Change

### Changed

### Reason

### Impact

### Issue Changes
- Added #...
- Deferred #...
- Replaced #...
```

Do not silently alter expectations.

## Phase Completion Update

Before final merge:

```markdown
## Phase Completion

### Delivered
- ...

### Child Issues
Completed: X
Deferred: Y
Cancelled: Z

### Validation
- unit tests: passing
- integration tests: passing
- lint: passing
- type checks: passing

### Audit
P0: 0
P1: 0
Open P2: ...

### Documentation
- ...

### Ready for Main
Yes
```
