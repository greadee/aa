# github-os — Module Update Plan: issues, sub-problems, and module updates

> Authored at the **start** of the module update, on branch `ph0-scaffold`, which owns the GitHub OS.
> Scope: the durable process surface in `docs/github-os/`.

## Objective

Record the issue, problem, and sub-problem workflow that the phase 1 and phase 2 module updates used in practice — umbrella problem documents, one sub-problem per owning branch, and module update plans and summaries — as a first-class part of the GitHub OS. After this update an agent can decompose a cross-branch problem, assign each sub-problem to the branch that owns its module, and deliver it as a module update without reconstructing the process from examples.

## Starting State

- Starting ref: `main @ 79b495d`; branch `ph0-scaffold` fast-forwarded to it.
- Available: the GitHub OS modules for phase management, phase documentation, sprint planning, issues and stories, pull requests, code review, audits, reconstruction, ADRs, documentation, Git workflow, and model selection; the phase, github, and feature templates; the `docs/index/` and `docs/issues/` conventions.
- Missing: any GitHub OS module for problem decomposition, sub-problem ownership, umbrella documents, and the module update cycle; templates for those documents.
- Known constraints:
  - `docs/github-os/` is canonical process documentation, used by agents and by `forge` template rendering.
  - The GitHub OS is owned by `ph0-scaffold` (Phase 0 — Foundation & Governance), whose retained branch is reused for this update.
  - `docs/issues/` durable issue documents are introduced by the phase 1 and phase 2 module updates; this update documents the convention without duplicating those documents.

## Scope

### In Scope
- `docs/github-os/issues-and-subproblems.md`: decomposition, sub-problem ownership, issue documents, dependency order.
- `docs/github-os/module-updates.md`: the module update cycle, plan, summary, slices, and pull request.
- Templates under `docs/github-os/templates/issues/` and `docs/github-os/templates/module/`.
- Wiring in the GitHub OS README, `AA.md`, `docs/github-os/documentation-system.md`, and `MANIFEST.json`.
- A module update plan and summary for this change.

### Out of Scope
- The `docs/issues/` documents themselves, which are delivered by the phase 1 and phase 2 sub-problems.
- Changes to `forge` behavior or its template rendering.
- Any new phase; this is a governance update on an existing branch.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P0U-001 | Model a large deficiency as an umbrella problem with distinct sub-problems | One issue cannot span several modules and branches without collapsing ownership; separate problems stay separately trackable |
| ADR-P0U-002 | Each sub-problem is owned by exactly one branch — the phase branch of its module | Matches the existing branch/folder ownership rule and the phase 1 and phase 2 practice |
| ADR-P0U-003 | Work for a merged phase reopens its retained branch as a module update; no new phase | Avoids phase proliferation and keeps module history with the module (git-workflow branch retention) |
| ADR-P0U-004 | Module update plans and summaries mirror phase plans and summaries beside the module | One recognizable lifecycle, reusing the plan-before/summary-after discipline |
| ADR-P0U-005 | Durable issue documents live under `docs/issues/`; GitHub stays the execution record | Consistent with the documentation system: knowledge is durable, execution is live |

## Slices

Each slice maps to exactly one commit.

| Slice | Goal | Commit message |
|---|---|---|
| 1 | Issue, sub-problem, and module update documentation and this plan | `add github os issue and subproblem workflow` |
| 2 | Umbrella, sub-issue, module plan, and module summary templates | `add github os module update templates` |
| 3 | README, router, documentation tree, and manifest wiring | `wire github os issue workflow into indexes` |
| 4 | Module update summary | `add github os module update summary` |
| 5 | Link the module update pull request | `link github os module update pull request` |

## Required End State

- [ ] `docs/github-os/issues-and-subproblems.md` documents umbrella problems, sub-problems, and ownership.
- [ ] `docs/github-os/module-updates.md` documents the module update cycle and artifacts.
- [ ] Templates exist for the umbrella document, the sub-problem document, the module update plan, and the module update summary.
- [ ] The GitHub OS README, `AA.md`, `documentation-system.md`, and `MANIFEST.json` link the new modules.
- [ ] A module update summary is authored and links the pull request.

## Exit Criteria

- [ ] The workflow is documented and linked from the GitHub OS index and the task router.
- [ ] The templates render as readable skeletons and match the worked examples.
- [ ] The docs link check passes.
- [ ] A module update PR is opened on `ph0-scaffold` into `main`.

## Test Plan

| Layer | What is tested |
|---|---|
| links | every relative link in the new and edited documents resolves |
| consistency | templates match the documented structure and the phase 1/2 examples |
| index | the new modules appear in `docs/github-os/README.md`, `AA.md`, and `MANIFEST.json` |
| spelling | document titles and cross-references use the established terminology |
