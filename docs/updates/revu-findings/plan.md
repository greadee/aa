# Review Findings Remediation Sprint (`revu-findings`) — Plan

> **Type:** update phase (`revu-findings`). **Branch:** `dev`. **Source:** PR #21 review, consolidated in issue [#36](https://github.com/greadee/aa/issues/36).
> **Finding count:** 77 (fail-open 4, orphan/wiring 23, stale-reference 50). Archived findings: `review-artifacts` branch, `runs/pr@df8792f2.md`.
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Resolve the review findings the review engine raised on the `issue-impl-sep28` sprint (PR #21), in
module-grouped stages, so that a follow-up `revu` audit of `dev` reports no findings — or documents
each accepted exception — and the held-open sprint issues (#23, #24, #25, #27, #28, #29, #31, #32)
can be reconciled.

## Starting State

- Starting ref: `dev @ df8792f2` (PR #21 head). `main @ bf8e09d`.
- The ten sprint issues were delivered; a `revu` review of `dev` vs `main` produced 77 findings, and
  the findings-aware reconcile reopened the eight issues whose areas have findings.
- No production composition root exists: the runtime/kernel/toolbox features are exercised only by
  tests (this is the single largest finding class: 23 orphaned exported symbols).

## Scope

### In scope
- **Fail-open guards (4).** Make each nil-guarded check fail closed.
- **Orphan wire-up / dead code (23).** Provide a composition root and wire the test-only constructors,
  or remove genuinely dead exported API.
- **Stale references (50).** Fix genuine stale doc/path references; add detector ignore patterns for
  template placeholders and module-relative examples (triaged, not blanket-suppressed).
- Re-run `revu` to verify, post the audit summary, and reconcile the affected issues.

### Out of scope
- New features, UI, and anything not tied to a finding above; raise new issues instead.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| RD-1 | Fail closed: a guard whose dependency is absent returns the error (deny), not open | A missing policy/enforcer must never silently disable a check |
| RD-2 | Introduce a single composition root (`cmd/aa` + `internal/bootstrap`) that wires the built features | The orphan findings all stem from "no production caller"; one root resolves them coherently |
| RD-3 | Wire-in where the feature is real; delete exports that are genuinely dead | Do not keep test-only API as a substitute for wiring |
| RD-4 | Triage stale references: fix genuine ones; add `ignore` for template placeholders and intentional module-relative examples | The detector has known false positives (template filenames, doc-relative snippets) |
| RD-5 | One commit per stage/module grouping; stop at each stage boundary | Keeps each commit a single, describable change |

## Stages and commits

**DoD per commit:** affected module `go build`/`go vet`/`go test` green; `gofmt` clean; `tools/archtest`
clean; docs link check valid.

### Stage 0 — Fail-closed guards
- `runtime/sandbox/sandbox.go:139` — deny when the policy is nil (or require a policy at construction).
- `kernel/allocator/routing/routing.go:95` — fail closed on over-budget when a budget is set.
- `memory/retrieval/retrieve.go:107` — a nil filter must not bypass filtering semantics.
- `memory/retrieval/trace.go:99` — treat a nil `Redacted` as not-redacted explicitly (no silent pass).
- **Commit** `fail closed on nil dependencies in sandbox, routing and retrieval`.

### Stage 1 — Composition root & wiring (orphan findings)
- Add `cmd/aa` + `internal/bootstrap` composing the delivered seams: kernel (context, allocator,
  routing, scheduler, joblearn, observability), runtime (sandbox, intake), toolbox (computeruse),
  registry (models), memory (retrieval/retention/promotion), forge (memsync).
- Wire each orphaned constructor into the root; delete exports confirmed dead
  (`memory/projection` unused function).
- Grouped commits by module area: `wire kernel seams into a composition root`,
  `wire runtime, toolbox and registry into the composition root`,
  `wire memory and forge into the composition root and drop dead exports`.

### Stage 2 — Documentation hygiene (stale-reference findings)
- Fix genuine stale references: `contracts/POLICY.md`, `docs/adr/*` (v1 paths), `docs/modules/**`
  (removed paths), `docs/phases/**`.
- Add `ignore` entries for template placeholders (`plan.md`, `summary.md`, `issues/issue-123-name.md`)
  and intentional module-relative examples.
- **Commit** `repair stale documentation references after the v1 sunset`.

### Stage 3 — Verification & closeout
- Re-run `revu run --profile audit` on `dev`; attach the summary; reconcile affected issues.
- Author `summary.md` + `summary.uml`; close #36 when the audit is clean (or list accepted exceptions).
- **Commit** `add revu-findings sprint summary and reconcile issues`.

## Finding → stage map

| Finding class | Count | Stage |
|---|---|---|
| `detector.fail-open` | 4 | Stage 0 |
| `detector.orphan` (test-only wiring) | 22 | Stage 1 |
| `detector.orphan` (unused export) | 1 | Stage 1 |
| `detector.stale-reference` | 50 | Stage 2 |

## Validation gates

| Gate | Command |
|---|---|
| Formatting | `gofmt -l .` |
| Go | per-module `go build/vet/test ./...` |
| Boundaries | `tools/archtest -root ..` (from `tools`) |
| Contracts | `contracts` Go tests; `contracts/python` unittest; `contracts/typescript` `tsc --noEmit` |
| Inference | `runtime/inference` pytest, `ruff`, `mypy` |
| Docs | fence-aware link scan |
| Review | `revu run --profile audit` → 0 findings (or documented exceptions) |

## Exit criteria

- [ ] The four fail-open guards fail closed, with tests.
- [ ] A composition root exists and every orphaned symbol is wired or removed.
- [ ] Stale references repaired; remaining detector output explained by documented ignore patterns.
- [ ] `revu` audit of `dev` reports 0 findings, or each accepted finding is recorded.
- [ ] Sprint summary + UML authored; #36 closed; affected issues reconciled.
