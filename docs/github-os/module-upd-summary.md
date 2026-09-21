# github-os — Module Update Summary: issues, sub-problems, and module updates

> Authored at the **end** of the module update, on branch `ph0-scaffold`. Derived from [module-upd-plan.md](module-upd-plan.md).
> Phase PR: linked in the follow-up slice.

## Delivered

| Slice | Status | Commit message | Notes |
|---|---|---|---|
| Issue and sub-problem workflow | Complete | `add github os issue and subproblem workflow` | `issues-and-subproblems.md`, `module-updates.md`, this plan |
| Module update templates | Complete | `add github os module update templates` | `templates/issues/`, `templates/module/` |
| Index and router wiring | Complete | `wire github os issue workflow into indexes` | README, `AA.md`, `documentation-system.md`, `MANIFEST.json` |
| Module update summary | Complete | `add github os module update summary` | this file |
| Phase PR link | Complete | `link github os module update pull request` | follow-up commit |

## What was added

- `docs/github-os/issues-and-subproblems.md` — problem versus sub-problem, umbrella documents, the one-branch ownership rule, sub-problem documents under `docs/issues/`, dependency order, and the agent discovery policy.
- `docs/github-os/module-updates.md` — when to use a module update, the reopen-to-merge cycle, the module update plan and summary, slices, the pull request, and branch retention.
- `docs/github-os/templates/issues/umbrella.md`, `docs/github-os/templates/issues/sub-issue.md`, `docs/github-os/templates/module/module-upd-plan.md`, and `docs/github-os/templates/module/module-upd-summary.md`.
- Wiring in the GitHub OS README module table, templates list, and core rules; the `AA.md` task router; the `documentation-system.md` tree; and `MANIFEST.json`.

## What this changes about the module and the app

- The GitHub OS gains two modules that make an existing, exercised workflow explicit: a cross-branch problem is recorded as an umbrella, split into sub-problems, and each sub-problem is delivered on the branch that owns its module.
- The documentation tree now names `docs/issues/` and the module update plan and summary beside a module.
- The app is otherwise unchanged: no module code, contracts, or runtime behavior is touched.

## Is this part of a larger change?

No. This is a governance update on the phase 0 branch that owns the GitHub OS; it is not a sub-problem of a separate umbrella. It documents the workflow exercised by the phase 1 and phase 2 module updates (PRs [#12](https://github.com/greadee/aa/pull/12) and [#13](https://github.com/greadee/aa/pull/13)), which are sub-problems of the umbrella recorded on those branches.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Links | Pass | relative-link scan over 99 Markdown files resolves every target |
| Index | Pass | the new modules appear in `README.md`, `AA.md`, `documentation-system.md`, and `MANIFEST.json` |
| JSON | Pass | `MANIFEST.json` parses |
| CI | Pending | docs link check runs on the pull request |

## Decisions affirmed

| # | Decision | Outcome |
|---|---|---|
| ADR-P0U-001 | Umbrella problem with distinct sub-problems | Affirmed — matches the exercised phase 1/2 pattern |
| ADR-P0U-002 | One branch owns each sub-problem | Affirmed |
| ADR-P0U-003 | Reopen a merged phase branch as a module update | Affirmed — the phase 1 and phase 2 updates did exactly this |
| ADR-P0U-004 | Module update plan and summary mirror phase plan and summary | Affirmed |
| ADR-P0U-005 | Durable issue documents under `docs/issues/` | Affirmed — consistent with the documentation system |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Link the sub-problem document and `docs/issues/README.md` from the new modules | Referenced as paths in prose | `docs/issues/` is introduced by the unmerged phase 1 and phase 2 branches; hard links would fail the link check on this branch |

## Follow-Up

- When the phase 1 and phase 2 module updates merge, the `docs/issues/` documents become durable on `main`; the GitHub OS modules already reference them by path.
- `forge` template rendering may load the new `templates/issues/` and `templates/module/` skeletons; wiring that is out of scope for this governance update.
