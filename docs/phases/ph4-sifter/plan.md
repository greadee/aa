# Phase 4 — Sifter: Plan

> Authored at the **start** of the phase. Branch and folder: `ph4-sifter`.

## Objective

Deliver `aa-sifter` as the single governed path to every model. Migrate the existing `compute-sifter` Python project into the monorepo as a `src/aa_sifter` package, expose the stable contracts RPC surface (`sifter.route`, `sifter.generate`, `sifter.health`), and close the governance gaps that the migration audit found: redaction on **every** cloud path, enforced token limits, wired budget waivers and standing-rule grants, hardware/recommender reuse, and memory-bound traces. The sifter holds no project state and performs no orchestration; decisions are deterministic and human gates stay authoritative.

## Starting State

- Starting ref: `main @ ba195e1` (after phase 3 merge).
- Available:
  - `aa-contracts` v1 schemas and Python bindings, including `route_request`/`route_response`, `execution_contract` (budget, capabilities), and `memory_record` with lifecycle (`contracts/python/aa_contracts`, `contracts/schemas/v1`).
  - RPC v1 spec naming `sifter.route`, `sifter.generate`, `sifter.health` (`contracts/rpc/rpc-v1.md`).
  - A scaffolded `sifter/` module: `pyproject.toml` (hatchling, `src/aa_sifter`), `README.md`, directive `aa-sifter.md`.
  - The complete `compute-sifter` Python project at `C:\Users\prool\compute-sifter` (63 package files, ~60 test files, 326 passing tests): classifier, approval gate, routing policies, budgets, providers (Ollama/OpenAI-compatible/DeepSeek), profiles/config, catalog, hardware detection, recommender, task DAG, verification, history, metrics, CLI, and a local desktop surface.
- Missing:
  - Any `aa_sifter` code; `sifter/src/` does not exist.
  - The contracts bridge and RPC service.
  - Redaction on most cloud paths (only two handoff sites redact today).
  - Enforced token limits, live budget waivers/rule grants, and any memory sink.
  - A Python test/lint job for `sifter` in `.github/workflows/ci.yml`.
- Known constraints:
  - Python module must not hold project state or orchestrate; `kernel` reaches it over RPC.
  - Routing is deterministic and brand-free; provider/model names live in config/catalog only.
  - Major and critical decisions are forced through human approval; relaxing security is itself major.
  - No cloud call may carry unredacted text; no test in the default suite may touch the network.
  - The sifter must not import other aa modules; it only validates against `aa_contracts`.

## Scope

### In Scope
- Migrate `compute-sifter` into `sifter/src/aa_sifter` and `sifter/tests`, renamed and src-laid-out, with packaging, CLI, and desktop surfaces.
- A contracts bridge mapping `route_request`/`route_response` and `memory_record` to sifter types via `aa_contracts`.
- A stable, transport-agnostic JSON-RPC service implementing `sifter.route`, `sifter.generate`, and `sifter.health` over the v1 envelope.
- One outbound redaction chokepoint so every expert/cloud request is redacted before it leaves the process.
- Enforced token limits: context sizing against `context_limit` and `max_output_tokens` passed to providers.
- Budget enforcement with pre-flight estimates plus wired cost waivers and standing-rule grants (for approval and debugging escalation).
- A `MemorySink` seam that emits `memory_record`-shaped trace candidates; default no-op.
- Hardware detection and the deterministic recommender retained and exposed through the library and CLI.
- A `sifter` CI job (format, lint, typecheck, tests) and packaging checks.

### Out of Scope
- Real cross-process transport binding (named pipe/socket server hosting) and process supervision; the service is in-process and transport-agnostic.
- Live cloud calls in CI; the live smoke workflow stays manual and opt-in.
- Learned routing or embeddings; routing stays deterministic.
- Kernel-side wiring of a production runtime adapter (the kernel adapter change lands when the sifter service is hosted).
- Storage of traces in `aa-memory`; the sink interface and record shape land here, the writer lands in a later phase.

### Required End State
- [ ] `pip install -e ".[dev]"` works in `sifter/`; imports resolve under `aa_sifter`.
- [ ] `sifter.route`, `sifter.generate`, and `sifter.health` round-trip against `aa_contracts` request/response shapes.
- [ ] Every expert-tier outbound request passes through the secret redactor; a spy provider sees no secret material.
- [ ] Messages are sized to the tier `context_limit` and `max_output_tokens` is passed to providers.
- [ ] A standing-rule grant and a cost waiver each change a major/cloud decision deterministically, with tests.
- [ ] A memory candidate is emitted per run through a `MemorySink`.
- [ ] `ruff format --check`, `ruff check`, `mypy`, and `pytest` pass; no default test uses the network.
- [ ] `go build`/`vet`/`test`, `gofmt`, and boundary checks remain green.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P4-001 | Migrate by copy-then-cutover; keep class names (`ComputeSifter`) and add a `Sifter` alias | Preserves a proven, tested implementation and avoids an unnecessary rewrite; the package/distribution are renamed to `aa_sifter`/`aa-sifter` |
| ADR-P4-002 | Keep the deterministic classifier/preflight/policy engine as the routing authority; models never decide the route | Reproducibility and safety; matches the kernel's deterministic control-plane ADR |
| ADR-P4-003 | One outbound redaction chokepoint in `_call_tier` for every expert-tier request | A single enforced seam is auditable and testable; per-caller redaction proved incomplete |
| ADR-P4-004 | Token limits are enforced, not configured: size messages to `context_limit`, pass `max_output_tokens` | Prevents silent over-limit cloud sends and runaway output |
| ADR-P4-005 | Budget waivers and rule grants are wired to the policy engine and budget tracker | The capability existed as dead code; least-privilege plus explicit, recorded waivers is the safe shape |
| ADR-P4-006 | The RPC service is transport-agnostic JSON-RPC 2.0 over the v1 envelope | Lets the kernel host it over the local socket without coupling the service to transport |
| ADR-P4-007 | Contracts are validated with `aa_contracts` at the RPC boundary; sifter types never leak cross-module | One schema source; the Python binding is the only cross-module dependency |
| ADR-P4-008 | Traces are emitted as `memory_record` candidates through a `MemorySink` | Evidence-gated, lifecycle-aware memory; the sifter never writes canonical state |
| ADR-P4-009 | `generate` still governs expert tier: redaction and budget apply even though approval is the caller's contract | Cloud egress must be safe regardless of caller intent |
| ADR-P4-010 | Default tests make zero network calls; live smoke stays manual | CI safety and cost control |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase and its intended infrastructure. **Inputs** — architecture, contracts, prior phase summaries. **Expected Output** — `docs/phases/ph4-sifter/plan.md` + `plan.uml`. **Model Class** — Strong. **Commit Message** — `add ph4 sifter phase plan`. **Validation** — docs link check. **Dependencies** — none. **Documentation** — this file and `plan.uml`.

### Slice 2 — Migrate core config, models, catalog, and system
**Goal** — Land the renamed package skeleton and the provider/config/catalog/hardware layers. **Inputs** — `compute-sifter/sifter/{config,models,catalog,system,model_defaults.py}`. **Expected Output** — `sifter/src/aa_sifter/...` and updated `sifter/pyproject.toml`. **Model Class** — Lightweight. **Commit Message** — `migrate sifter core config models and providers`. **Validation** — `python -c "import aa_sifter.models"`. **Dependencies** — slice 1. **Documentation** — package layout in `sifter/README.md`.

### Slice 3 — Migrate routing, context, metrics, and verification
**Goal** — Land preflight/policy/budget/escalation, redaction/compression/handoff, usage/trace, and verifiers. **Inputs** — corresponding `compute-sifter/sifter` subpackages. **Expected Output** — code + import rewrite. **Model Class** — Lightweight. **Commit Message** — `migrate sifter routing context and verification`. **Validation** — import smoke. **Dependencies** — slice 2. **Documentation** — README module list.

### Slice 4 — Migrate decisions, tasks, history, and the app
**Goal** — Land classifier/gate/approval/rules, the task DAG, SQLite history, and `ComputeSifter`. **Inputs** — corresponding subpackages. **Expected Output** — code + import rewrite. **Model Class** — General. **Commit Message** — `migrate sifter decisions tasks and app`. **Validation** — construct `ComputeSifter` with fakes. **Dependencies** — slice 3. **Documentation** — README flow.

### Slice 5 — Migrate CLI and desktop surfaces
**Goal** — Land the CLI, doctor, and local desktop service/server/launcher/web assets, de-branded to `aa-sifter`. **Inputs** — `compute-sifter/sifter/{cli,desktop,doctor.py}`. **Expected Output** — code, web assets, `sifter/docs/desktop.md`. **Model Class** — Lightweight. **Commit Message** — `migrate sifter cli and desktop surfaces`. **Validation** — `aa_sifter.cli.main --help`. **Dependencies** — slice 4. **Documentation** — `sifter/docs/desktop.md`.

### Slice 6 — Port the test suite
**Goal** — Move all tests to `sifter/tests` with the new package/module paths and make the default suite green. **Inputs** — `compute-sifter/tests`. **Expected Output** — `sifter/tests/**` + pytest config. **Model Class** — General. **Commit Message** — `port sifter tests to aa-sifter`. **Validation** — `pytest`. **Dependencies** — slice 5. **Documentation** — README test section.

### Slice 7 — Python CI job and packaging
**Goal** — Add a `sifter` job to CI (Python 3.12/3.13: install, ruff, mypy, pytest) and a package-build check. **Inputs** — `.github/workflows/ci.yml`, `pyproject.toml`. **Expected Output** — CI job + build data config. **Model Class** — Lightweight. **Commit Message** — `add sifter python ci job`. **Validation** — workflow parses; local install succeeds. **Dependencies** — slice 6. **Documentation** — `sifter/README.md`.

### Slice 8 — Contracts bridge and RPC service
**Goal** — Map `route_request`/`route_response` via `aa_contracts` and implement `sifter.route`, `sifter.generate`, `sifter.health` over the v1 envelope. **Inputs** — `contracts/python/aa_contracts`, `contracts/rpc/rpc-v1.md`. **Expected Output** — `aa_sifter/contracts.py`, `aa_sifter/rpc/*.go`-equivalent (`service.py`, `envelope.py`) + tests. **Model Class** — Strong. **Commit Message** — `add sifter rpc service and contract bridge`. **Validation** — contract round-trip tests; determinism test. **Dependencies** — slice 7. **Documentation** — README interfaces.

### Slice 9 — Redaction chokepoint
**Goal** — Redact every expert-tier outbound request in one place; cover `generate`, preflight, and decomposition. **Inputs** — `context/handoff.py`, `app.py`, `routing/preflight.py`, `tasks/decomposition.py`. **Expected Output** — centralized redaction + security tests with a spy provider. **Model Class** — General. **Commit Message** — `enforce sifter cloud redaction`. **Validation** — no secret reaches an expert provider; local path unchanged. **Dependencies** — slice 8. **Documentation** — README security section.

### Slice 10 — Token limits and budget waivers/grants
**Goal** — Size messages to `context_limit`, pass `max_output_tokens`, estimate cost/tokens pre-flight, and wire `cost_waiver`/`has_grant` into approval and escalation. **Inputs** — `context/compression.py`, `routing/budget.py`, `decisions/policy.py`, `app.py`. **Expected Output** — enforced limits + wired waivers + tests. **Model Class** — Strong. **Commit Message** — `add sifter token limits and budget waivers`. **Validation** — truncation/limit tests; grant/waiver tests. **Dependencies** — slice 9. **Documentation** — README budgets/rules.

### Slice 11 — Memory traces and recommender wiring
**Goal** — Emit a `memory_record`-shaped candidate per run through a `MemorySink`; surface hardware/recommender through the library and health. **Inputs** — `metrics/trace.py`, `recommend/recommender.py`, `system/hardware.py`, `app.py`. **Expected Output** — `aa_sifter/memory.py` + tests; recommender tests retained. **Model Class** — General. **Commit Message** — `add sifter memory traces and recommender wiring`. **Validation** — sink receives a valid candidate; `aa_contracts.validate` accepts it. **Dependencies** — slice 10. **Documentation** — README memory/programmatic section.

### Slice 12 — Phase summary and UML
**Goal** — Record the as-built phase. **Inputs** — all prior slices. **Expected Output** — `summary.md`, `summary.uml`, index updates. **Model Class** — Strong. **Commit Message** — `add ph4 sifter phase summary`. **Validation** — link check. **Dependencies** — slice 11. **Documentation** — summary + indexes.

## Exit Criteria

- [ ] `aa_sifter` package installs and imports; CLI runs.
- [ ] The full ported default test suite passes and makes no network calls.
- [ ] `sifter.route`/`generate`/`health` validate against `aa_contracts` and are covered by tests.
- [ ] All expert-tier egress is redacted; tests prove it.
- [ ] Token limits and budgets are enforced, including waived/ granted paths.
- [ ] Memory candidates are emitted through a `MemorySink`.
- [ ] CI is green for the sifter job and the existing jobs.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | migration parity (config, routing, decisions, tasks), success |
| contract | `route_request`/`route_response` round-trip through `aa_contracts`; RPC envelope |
| integration | route→decision; generate→result; approval, budget, and escalation flows with fakes |
| security | redaction on every expert path; no secret reaches a spy provider; `local_only` yields zero expert traffic |
| determinism | stable route decisions and reasons for identical inputs |
| limits | context sizing to `context_limit`; `max_output_tokens` forwarded; budget estimates |
| memory | emitted candidate validates as a `memory_record` and carries provenance |
| boundary | `sifter` imports no aa module except the `aa_contracts` binding |
| live | one optional, manual, paid provider smoke (never in default CI) |
