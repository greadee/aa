# aa-sifter — module documentation

> Project history for the `aa-sifter` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-sifter.md`](../../../sifter/aa-sifter.md) · **Source** — [`sifter/`](../../../sifter/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Given a prompt and context, decide which model tier and provider should handle it, enforce budget and approval, execute, verify, and escalate on failure. The sifter is the sole path to any model.

## Owns

- Prompt decoding and decision classification (routine/significant/major/critical).
- The human approval gate for major and critical decisions.
- Preflight assessment and policy routing (`local_only`, `cloud_only`, `local_first`, `expert_first`, `adaptive`, `budget_constrained`).
- Budgets, escalation, and provider abstraction (Ollama, OpenAI-compatible, DeepSeek).
- Verification and context compression/handoff.
- Compute orchestration across local and cloud models.

## Must not

- Hold project state or orchestrate work.
- Send data to the cloud without redaction.
- Hard-code model names in routing logic.
- Bypass approval or budget.

## Interfaces

- RPC `route(request) -> decision`.
- RPC `generate(messages, tier) -> result`.
- Progress stream for the control plane.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`catalog (python)`](./catalog.md) | Advisory bundled metadata about known local/cloud models (never a whitelist). |
| [`cli (python)`](./cli.md) | ``aa-sifter serve``: host the RPC service over the per-user local socket. |
| [`config (python)`](./config.md) | Shipped reference profile built on the shared model defaults. |
| [`context (python)`](./context.md) | Deterministic context compaction/fitting, secret redaction, local-task handoff, and escalation packet. |
| [`decisions (python)`](./decisions.md) | Decision-significance classification, the human approval gate, approval providers, and standing-rule policy. |
| [`desktop (python)`](./desktop.md) | Local HTTP control-plane server, process launcher, task service, and path handling. |
| [`history (python)`](./history.md) | Durable run/approval/event/standing-rule persistence (SQLite). |
| [`metrics (python)`](./metrics.md) | Structured trace/event logging and token/cost usage accounting. |
| [`models (python)`](./models.md) | Provider abstraction, provider registry/factory, runtime model config, and Ollama/OpenAI-compatible/DeepSeek clients. |
| [`recommend (python)`](./recommend.md) | Deterministic hardware/goal-based profile and context recommendation. |
| [`routing (python)`](./routing.md) | Deterministic routing policies, preflight assessment, cloud budgets, and escalation evaluation. |
| [`rpc (python)`](./rpc.md) | aa-sifter RPC service (inter-module JSON-RPC v1). |
| [`schemas (python)`](./schemas.md) | Bundled JSON Schemas used to validate the RPC boundary. |
| [`system (python)`](./system.md) | Best-effort hardware detection (CPU/GPU/VRAM/RAM). |
| [`tasks (python)`](./tasks.md) | Task decomposition, DAG scheduling, concurrent execution, and the inference queue. |
| [`verification (python)`](./verification.md) | Post-generation verification strategies (null/command/file/callable/composite). |

