# Toolbox Pre-Hardening — Execution Guide

This document lists all files created and the exact `gh` commands to create GitHub issues, a milestone, and a PR.

## Files Created

### Issue Documents (knowledge record)
- `docs/issues/toolbox-pre-hardening.md` — Umbrella issue
- `docs/issues/toolbox-rpc-toctou-race.md` — P0: TOCTOU race
- `docs/issues/toolbox-stale-contract-version.md` — P1: Stale contract version
- `docs/issues/toolbox-rpc-typed-requests.md` — P1: map[string]any
- `docs/issues/toolbox-rpc-timeout.md` — P2: No timeout
- `docs/issues/toolbox-rpc-error-structure.md` — P2: Error structure
- `docs/issues/toolbox-workflow-config-validation.md` — P2: Untyped Config
- `docs/issues/toolbox-metrics-observability.md` — P2: No metrics
- `docs/issues/toolbox-contract-file-split.md` — P3: File split
- `docs/issues/toolbox-contracts-parity-review.md` — P3: Binding parity

### Sprint Plan
- `docs/updates/toolbox-pre-hardening/plan.md` — 9 slices, exit criteria

### PR Body
- `docs/updates/toolbox-pre-hardening/pr-body.md` — PR body template

### Milestone Description
- `docs/updates/toolbox-pre-hardening/milestone.md` — Milestone description

### Updated
- `docs/issues/README.md` — Index updated with new issues

## GitHub Commands

Run these commands in sequence. Replace `???` with the actual GitHub issue numbers after creation.

### Step 1: Create the umbrella issue

```bash
cd /Users/prool/aa
gh issue create \
  --title "Pre-hardening: toolbox module code review findings" \
  --label "audit" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-pre-hardening.md
```

Note the returned issue number (e.g., #33).

### Step 2: Create all sub-issues (linked to umbrella)

```bash
UMBRELLA=33  # Replace with actual issue number

gh issue create \
  --title "Fix RPC TOCTOU race condition in Service.Handle" \
  --label "bug,P0" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-rpc-toctou-race.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Update toolbox test fixtures to contract v2" \
  --label "bug,P1" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-stale-contract-version.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Replace map[string]any with typed RPC request structs" \
  --label "technical-debt,P1" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-rpc-typed-requests.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Honor TimeoutMS in RPC handler" \
  --label "bug,P2" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-rpc-timeout.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Add structured data to RPC error responses" \
  --label "technical-debt,P2" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-rpc-error-structure.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Validate WorkflowStep.Config per step kind" \
  --label "technical-debt,P2" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-workflow-config-validation.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Add metrics interface to toolbox" \
  --label "feature,P2" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-metrics-observability.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Split large contract files into domain-specific units" \
  --label "refactor,P3" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-contract-file-split.md \
  --reference "refs/$UMBRELLA"

gh issue create \
  --title "Review Python/TypeScript bindings for parity with Go" \
  --label "audit,P3" \
  --milestone "v1 — Toolbox Pre-Hardening" \
  --body-file docs/issues/toolbox-contracts-parity-review.md \
  --reference "refs/$UMBRELLA"
```

### Step 3: Create the milestone

```bash
# Create milestone (if not auto-created by gh issue --milestone)
gh milestone create \
  --title "v1 — Toolbox Pre-Hardening" \
  --description "$(cat docs/updates/toolbox-pre-hardening/milestone.md | sed -n '/^```/,/^```/p' | sed '1d;$d')" \
  --state open
```

### Step 4: Create the PR

```bash
# Create a branch for this sprint
git checkout -b update-toolbox-pre-hardening

# Commit all changes
git add docs/issues/toolbox-*.md docs/updates/toolbox-pre-hardening/ docs/issues/README.md
git commit -m "add toolbox pre-hardening issues, sprint plan, and PR body"

# Push and create PR
git push origin update-toolbox-pre-hardening

gh pr create \
  --title "add toolbox pre-hardening issues, sprint plan, and PR body" \
  --body-file docs/updates/toolbox-pre-hardening/pr-body.md \
  --base main \
  --label "audit"
```

### Step 5: Update issue references

After all issues are created, update the umbrella issue and sprint plan with actual issue numbers:

```bash
# Replace ??? with actual numbers in all documents
UMBRELLA=33  # umbrella issue number
P0=34        # TOCTOU race
P1A=35       # Stale contract version
P1B=36       # map[string]any
P2A=37       # TimeoutMS
P2B=38       # Error structure
P2C=39       # Untyped Config
P2D=40       # Metrics
P3A=41       # File split
P3B=42       # Binding parity

# Update umbrella issue body
gh issue edit $UMBRELLA --body "$(sed "s/#???/#$P0/g; s/#???/#$P1A/g; s/#???/#$P1B/g; s/#???/#$P2A/g; s/#???/#$P2B/g; s/#???/#$P2C/g; s/#???/#$P2D/g; s/#???/#$P3A/g; s/#???/#$P3B/g" docs/issues/toolbox-pre-hardening.md)"

# Update sprint plan
sed -i "s/#???/#$P0/g; s/#???/#$P1A/g; s/#???/#$P1B/g; s/#???/#$P2A/g; s/#???/#$P2B/g; s/#???/#$P2C/g; s/#???/#$P2D/g; s/#???/#$P3A/g; s/#???/#$P3B/g" docs/updates/toolbox-pre-hardening/plan.md

# Update PR body
sed -i "s/#???/#$UMBRELLA/g" docs/updates/toolbox-pre-hardening/pr-body.md
```

## Summary of Findings by Severity

| Severity | Count | Issues |
|---|---|---|
| P0 — Critical | 1 | TOCTOU race (security/correctness) |
| P1 — High | 2 | Stale contract version, RPC map[string]any |
| P2 — Medium | 4 | No timeout, error structure, untyped Config, no metrics |
| P3 — Low | 2 | File split, binding parity |
