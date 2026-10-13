# Review Findings Remediation Sprint (`revu-findings`) — Summary

> Authored at the **end** of the update phase (`revu-findings`). Derived from [plan.md](./plan.md).
> Branch: `dev`. Findings source: issue [#36](https://github.com/greadee/aa/issues/36) (PR #21 review).
> Diagram: [summary.uml](./summary.uml).

## Delivered

| Stage | Outcome | Commit(s) |
|---|---|---|
| 0 — Fail-closed guards | Sandbox denies when permissions are declared without a policy; routing budget gate restructured; retrieval filter and trace redaction nil-handling made explicit; test added | `e7166fd` |
| 1 — Composition root | New `app/` module (`github.com/greadee/aa/app`, `aa selfcheck`) wiring kernel/runtime/toolbox/registry/memory/forge/obsv seams; dead `projection.DigestOf` removed; registered in `go.work`, CI, `archtest`, `MANIFEST.json` | `e46db90`, `8743e2f`, `c1b6479` |
| 2 — Documentation hygiene | All stale references marked deprecated in place with Markdown strikethrough across 28 docs (never deleted) | `1469b35` |
| 3 — Closeout | This summary + UML; accepted exceptions recorded; issues reconciled | (this commit) |

Engine-side: the `review-engine` stale-reference detector now honours strikethrough deprecation and
recommends it (ADR-0007 there), commit `cd39792`.

## Results

PR-scoped review findings: **77 → 4** (all four accepted; see below).

| Detector | Before | After |
|---|---|---|
| `detector.fail-open` | 4 | 0 |
| `detector.orphan` | 23 | 4 (accepted) |
| `detector.stale-reference` | 77 (50 reported, cap) | 0 |

## Accepted exceptions

Tracked as their own GitHub issues so they are not forgotten in future development.

| Symbol | Why accepted |
|---|---|
| `kernel/joblearn/tracesource.NewStatic` | Cross-package test seam (fixed trace set for other modules' tests). |
| `kernel/joblearn/tracesource.NewFake` | Cross-package test double for the `Source` seam. |
| `toolbox/computeruse.NewFake` | Cross-package test double for the `Driver` seam. |
| `toolbox/computeruse.NewProvider` | Needs a real `computeruse.Driver`, not implemented; a test double must not be wired into production. |

## Deviations

- **Composition root location.** The entrypoint lives at `app/` (module `github.com/greadee/aa/app`),
  not the conventional `cmd/aa/`, to match the repository's per-domain-module layout.
- **Deprecation, not exemption or deletion.** Superseded references are struck through in place
  rather than ignored, edited away, or removed — records are never deleted.

## Validation

| Gate | Result |
|---|---|
| `gofmt` / `go build` / `go vet` / `go test` (affected modules + `app`) | pass |
| `tools/archtest -root ..` | pass (`app` as composition root) |
| `revu --profile pr` (scoped to the sprint) | 4 findings, all accepted |
| Docs link scan | unchanged (strikethrough preserves links as historical text) |

## Notes

A **whole-repo** `revu` audit still shows many pre-existing uncomposed libraries (`sync`, `obsv`,
`visualizer`, `forge` lifecycle/template, `kernel` planner/gate/backtest, …). Those predate this
sprint and are out of its PR-scoped scope; they are candidates for a follow-up "wire the runtime"
phase.
