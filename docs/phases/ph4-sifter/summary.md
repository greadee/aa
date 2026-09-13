# Phase 4 — Sifter: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `219bf72` | plan.md + plan.uml |
| Core config, models, catalog, system | Complete | `6e605b0` | `aa_sifter/{config,models,catalog,system}` |
| Metrics, history, verification | Complete | `81c1ff8` | `aa_sifter/{metrics,history,verification}` |
| Decisions and policy engine | Complete | `d41d1cd` | `aa_sifter/decisions` |
| Routing, context, tasks, app | Complete | `38c75f5` | `aa_sifter/{routing,context,tasks}`, `app.py` |
| CLI, desktop, recommender | Complete | `ffa0e73` | `aa_sifter/{cli,desktop,recommend}`, `doctor.py` |
| Test suite ported | Complete | `0dc9c9b` | `sifter/tests` |
| Python CI job and packaging | Complete | `7cde0bc` | `.github/workflows/ci.yml` |
| Stabilize desktop approval test | Complete | `5bd7c82` | deterministic approval helper |
| Redaction chokepoint | Complete | `d1c6e66` | `_call_tier`, `generate`, preflight, decompose |
| Token limits and budget waivers/grants | Complete | `62399ec` | context sizing, `max_tokens`, waivers, grants |
| RPC service and contracts bridge | Complete | `b5bdf01` | `aa_sifter/{contracts,rpc}` |
| Memory traces and recommender wiring | Complete | `9f6ea46` | `aa_sifter/memory.py`, `sifter.recommend` |
| Test fix found by CI | Complete | `aa739c2` | fake provider in the memory-sink test |
| Phase summary and UML | Complete | this commit | summary.md, summary.uml, plan ref correction |

- Starting ref: `main @ da00db0` (after phase 3 merge).
- Tracking issue: none (phase executed via the phase PR).
- Milestone: none (matches phases 0–3 on this repository).
- Phase PR: [#5](https://github.com/greadee/aa1/pull/5)
- Branch: `ph4-sifter` (retained after merge on `aa1`)
- Migrated source of record: the standalone `compute-sifter` project (63 package files, ~60 test files)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet / test | Pass | unchanged; runs in CI |
| Go formatting / boundaries | Pass | `gofmt`, `archtest` jobs green |
| sifter install | Pass | `pip install -e ".[dev]"` (hatchling) |
| sifter format / lint | Pass | `ruff format --check`, `ruff check` clean |
| sifter types | Pass | `mypy` — 68 source files, no issues |
| sifter tests | Pass | 358 tests; default suite makes no network calls |
| Contract round-trip | Pass | `sifter.route`/`generate`/`health` validate against `aa_contracts` |
| Redaction | Pass | spy provider sees no secret on any expert path (`tests/unit/test_redaction_paths.py`) |
| Token limits | Pass | messages fit `context_limit`; `max_output_tokens` forwarded |
| Budgets / waivers / grants | Pass | pre-flight estimates; cost waiver raises the cap; grant bypasses escalation approval |
| Memory candidates | Pass | each run emits a validated `memory_record` candidate |
| Packaging | Pass | wheel includes catalog data and desktop web assets; clean-install smoke |
| CI | Pass | `sifter (py3.12)`, `sifter (py3.13)`, `sifter (package build)` |
| Docs links | Pass | runs on CI after push |
| Phase audit | Complete | see Audit below |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P4-001 | Copy-then-cutover; keep class names, add `Sifter` alias | Affirmed | 63 files migrated with tests green; package/distribution renamed |
| ADR-P4-002 | Deterministic classifier/preflight/policy decides the route | Affirmed | `test_route_is_deterministic`; no model in the route call |
| ADR-P4-003 | One outbound redaction chokepoint | Affirmed | `_call_tier` redacts all expert/cloud messages; `generate`, preflight, and decompose covered |
| ADR-P4-004 | Token limits enforced, not configured | Affirmed | `fit_messages` bounds context; `max_tokens` reaches providers |
| ADR-P4-005 | Budget waivers and rule grants wired | Affirmed | waiver raises the cap; grant bypasses escalation approval |
| ADR-P4-006 | Transport-agnostic JSON-RPC 2.0 service | Affirmed | `SifterService.handle` takes/returns envelope objects only |
| ADR-P4-007 | Validate at the RPC boundary with `aa_contracts` | Affirmed | route request/response and memory candidates validate |
| ADR-P4-008 | Traces emitted as `memory_record` candidates via a sink | Affirmed | `MemorySink` seam; promotion stays in `aa-memory` |
| ADR-P4-009 | `generate` still governs expert egress | Affirmed | cloud-disabled and redaction enforced even in `generate` |
| ADR-P4-010 | Default tests make zero network calls | Affirmed | fakes and `MockTransport` only; live smoke remains manual |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Slice order: core → routing/context → decisions/tasks/app → surfaces | Core → metrics/history/verification → decisions → routing/context/tasks/app → surfaces | The import graph required a topological order so each commit is independently importable; commit messages were adjusted accordingly |
| 2 | One draft phase PR with a 12-slice plan | 14 implementation commits plus this summary | The migration was grouped so each commit validates on its own; an extra commit fixed a CI timing flake and another fixed a test that had reached the network |
| 3 | Redaction via per-call-site helpers | A single `_call_tier` chokepoint plus the direct structured-call paths | The audit showed per-site redaction was incomplete; one seam is auditable |
| 4 | Pre-flight budget estimates include projected output cost | Estimates cover input tokens and input cost only | Projected maximum output cost was too conservative and broke the documented "actual usage accumulates" budget semantics; adding it would change behavior beyond the phase |
| 5 | Hardware/recommender behind the RPC surface | Added additive `sifter.recommend`; `health` advertises it | The v1 spec lists route/generate/health; the extra method is backward-compatible and recorded here |
| 6 | Production runtime adapter and kernel-side hosting | Deferred; the service is in-process and transport-agnostic | Hosting and the kernel adapter are a later integration; the service shape is fixed now |
| 7 | `plan.md` names its starting ref | Corrected to `main @ da00db0` in the summary commit | The plan is a start-of-phase artifact; the correction keeps the record accurate for this repository |

## Deferred / Follow-Up

- Host the RPC service over the named pipe / Unix socket transport and wire the kernel's production runtime adapter.
- A memory-backed `MemorySink` that records candidates into `aa-memory` (the sink seam and candidate shape exist).
- Live provider smoke for `aa-sifter` (manual, paid) mirroring the old workflow.
- Full JSON Schema (not just binding) validation at the RPC boundary.
- Retire the standalone `compute-sifter` repository once `aa-sifter` is hosted by the kernel.
- Reduce the duplicated desktop surface: it migrated for continuity but should eventually be a thin client of the control plane.

## Metrics

- Commits: 14 implementation + this summary
- Packages added: `aa_sifter` (68 source files) with `contracts`, `rpc`, `memory` new modules
- Tests: 358 in the default suite (port + 32 new)
- ADRs: ADR-P4-001 … ADR-P4-010
- Pitfalls recorded: 2 (CI approval timing race; a network-dependent test found by CI)
- PRs: 1 phase PR (#5)
- Issues: none

## Audit

| Severity | Count | Notes |
|---|---|---|
| P0 | 0 | — |
| P1 | 0 | — |
| P2 | 0 | — |
| P3 | 2 | Deferred hosting and memory writer (documented above) |

Audit result: no blocking findings. Boundary rule holds: `aa_sifter` imports no aa module except the optional `aa_contracts` binding.

## Retrospective

### Repeat
- Migrating by copy-then-rename with a mechanical import rewrite kept a tested implementation intact.
- A single redaction chokepoint plus a spy provider made the security property testable in one place.
- `fit_messages` (compress then truncate) is deterministic and easy to test.

### Avoid
- Keep commit boundaries aligned with the import graph when migrating a package; a cosmetic slice order produced unimportable intermediate commits before it was corrected.
- Timing-sensitive integration tests that sleep between approvals are flaky under CI load; poll for the next approval stage and retry the post.
- Do not shadow an existing method with an instance attribute (`cost_waiver`); rename the attribute (`waiver_ceiling`).

### Pitfalls

| Pitfall | Cause | Solution | Regression protection |
|---|---|---|---|
| Desktop major-approval integration test flaked on CI py3.12 | Fixed `sleep` between the two approval stages raced the service's approval future | Poll for the approval `stage` and retry the post until accepted | `tests/integration/test_desktop_workflows.py` helpers; 10× local stress run |
| A new test used the real provider on CI | A `ComputeSifter` was built without an injected provider, so it constructed Ollama and attempted egress | Inject `FakeProvider` | Default suite makes zero network calls; the failure surfaced on CI, not locally |

### As-Built Diagram

[summary.uml](./summary.uml)
