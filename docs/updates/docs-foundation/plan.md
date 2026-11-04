# v1 docs foundation — sprint plan

> **Type:** update phase (`docs-foundation`). **Branch:** `docs/v1-foundation` based on `dev`.
> **PR:** #93 against `dev`. **Status:** complete — see [summary.md](summary.md).

## Objective

Make the repository understandable and runnable for a new engineer in under five
minutes, and make the GitHub workflow enforceable through finished definitions of
done and templates. This is a foundation pass, not the final docs state; Phase 12
and Phase 13 still own security docs, release docs, `CHANGELOG.md`, and final
completion.

This sprint also closes the deferred contract item `ISS-CONTRACTS-1` (#76):
field-level cross-language binding parity, not name-only parity.

## Starting state

- The control-plane daemon, console surfaces, scheduler, runtime, memory, toolbox,
  sync, observation, and visualizer cores are delivered and composed in `app/`.
- The README still says "Pre-implementation" and has no quickstart.
- Issue and PR definitions of done are still drafts.
- GitHub has no issue templates, PR template, or `CONTRIBUTING.md`, and no license.
- `docs/` has good records but no clear audience entry, no current-state page, and
  several stale claims from earlier phases.
- `contracts/tools/check_binding_parity.py` still compares names only; known field
  drift exists between Go, TypeScript, and Python.

## Decisions

- License: MIT (`LICENSE.md`).
- Consolidate by reindexing and reconciling; preserve historical phase/update
  records and add dated supersession notes instead of deleting history.
- Media: generated deterministic captures are committed; owner-branded captures
  remain an explicit owner action.
- Homepage: `https://github.com/greadee/aa` until Phase 13.
- Current architecture diagrams describe the as-built v1 composition, not the
  target roadmap.

## Slices and issues

| Slice | Issue | Goal |
|---|---|---|
| 1 | #85 `ISS-DOCS-1` | Finalize issue/PR DoD, templates, comment templates, `CONTRIBUTING.md` |
| 2 | #86 `ISS-DOCS-2` | MIT license and repository metadata |
| 3 | #87 `ISS-DOCS-3` | README rewrite |
| 4 | #88 `ISS-DOCS-4` | Docs entry, audience paths, current state and limitations |
| 5 | #89 `ISS-DOCS-5` | Current architecture and data-flow diagrams |
| 6 | #90 `ISS-DOCS-6` | Media integration and validation |
| 7 | #91 `ISS-DOCS-7` | Stale-claim reconciliation |
| 8 | #92 `ISS-DOCS-8` | Docs review and closeout |
| contract | #76 `ISS-CONTRACTS-1` | Field-level binding parity gate |

## Validation

- Docs link check (`lychee --offline --include-fragments`).
- README quickstart from a clean state.
- Issue/PR template rendering.
- `python3 contracts/tools/check_binding_parity.py` and its self-tests.
- Post-docs audit recorded on the PR.

## Exit criteria

- [x] Every Sprint 3 issue has acceptance evidence.
- [x] New engineer can run `aa selfcheck` and the console in under five minutes.
- [x] Media and architecture diagrams are present.
- [x] Known limitations and current state are accurate.
- [x] Templates and DoD are complete and linked.
- [x] License, contributing, and repository metadata are set.
- [x] Docs indexes are consolidated and stale claims reconciled.
- [x] Contract binding parity checks fields, types, optionality, and direction.
- [x] PR reviewed, post-docs audit signed, and merged to `dev`.
