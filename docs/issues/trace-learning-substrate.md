# ISS-TRACE-LOOP — Observation and learning substrate is under-delivered

**Type:** technical debt / investigation
**Status:** open (umbrella)
**Affects branches:** `ph1-contracts`, `ph2-memory`, `ph3-kernel`, `ph4-sifter`, `ph9-joblearn`
**Related architecture:** D8 (separate operational telemetry, work history, memory), D10 (`obsv` hosted by kernel), D12 (deterministic role selection), D13 (all model access via `sifter`), §8 (memory hierarchy), §11 phases 2/3/4/9

## Problem

`aa` claims that every unit of work is *observable, replayable, attributable, and promotable into institutional memory*, and that the platform improves from its own history. Phases 0–9 were merged, but the substrate that makes this true was deferred rather than delivered:

- the work-observation module (`obsv`) is still a scaffold with no exported API;
- there is no cross-module contract for a per-step **trace** (only per-attempt telemetry summaries), so nothing downstream can store, compare, or learn from step-level evidence;
- the kernel host, its control-plane server, and a supervised end-to-end vertical slice were not delivered, so the modules are libraries behind fakes;
- production model routing (`sifter` over RPC) was replaced by seams and fakes;
- the job-learning loop therefore consumes thin summaries, not traces, and does not yet produce or evaluate improved subagent artifacts.

The consequence for the product goal — improving subagents from telemetry of prior work sessions across projects — is that there is no durable, comparable, redacted trace substrate to learn from.

## Scope boundary

This umbrella records the problem, the distinct sub-issues, and the branch dependency order. It intentionally does **not** specify solutions for branches whose work has not started. Each sub-issue's plan is authored with the pull request on the branch that owns it.

## Sub-issues

These are deliberately separate problems and are not one issue.

| Id | Problem | Module / branch | Why it matters | Status | Document |
|---|---|---|---|---|---|
| ISS-TRACE-1 | No cross-module contract for bounded per-step traces | `contracts` / `ph1-contracts` | Every later step needs one shared, versioned trace shape | active | [ph1-contracts-trace-contract.md](ph1-contracts-trace-contract.md) |
| ISS-TRACE-2 | No trace persistence, projection, promotion, or retention | `memory` / `ph2-memory` | Traces must become canonical, rebuildable knowledge before learning can use them | planned | (authored with the `ph2-memory` PR) |
| ISS-OBSV-1 | `obsv` live protocol missing; kernel host and control-plane server not delivered; no supervised vertical slice | `kernel` (+ `obsv`) / `ph3-kernel` | Observation is the source of every trace; without it the data is synthetic | planned | (authored with the `ph3-kernel` PR) |
| ISS-SIFTER-1 | Production `sifter` RPC wiring and cloud redaction deferred behind fakes | `sifter` / `ph4-sifter` | Distillation and any model-backed step must route through one governed, redacting boundary | planned | (authored with the `ph4-sifter` PR) |
| ISS-LEARN-1 | Learning loop does not distill or evaluate subagent artifacts from traces | `kernel/joblearn` / `ph9-joblearn` | The product's stated purpose — improving subagents from prior sessions — is unimplemented | planned | (authored with the `ph9-joblearn` PR) |

## Dependency order

```text
ISS-TRACE-1 (contracts trace object)
    -> ISS-TRACE-2 (memory trace store)
    -> ISS-OBSV-1 (kernel host + live observation + control-plane server)
    -> ISS-SIFTER-1 (governed model routing + redaction)
    -> ISS-LEARN-1 (distillation + evaluation over traces)
```

`ISS-OBSV-1` and `ISS-SIFTER-1` are independent of each other in mechanism but both are prerequisites for `ISS-LEARN-1`. `ISS-TRACE-1` unblocks storage and capture; the remaining problems are solved on their own branches.

## Ownership

No phase is added for this work. Each problem is completed on the module phase that already owns it, with the phase's retained branch reopened for a module-update PR.
