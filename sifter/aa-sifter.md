# aa-sifter.md

Agent directive for the `sifter` module (aa-sifter). This is the only Python module.

## Responsibility

Given a prompt and context, decide which model tier and provider should handle it, enforce budget and approval, execute, verify, and escalate on failure. The sifter is the sole path to any model.

## Owns

- Prompt decoding and decision classification (routine/significant/major/critical).
- The human approval gate for major and critical decisions.
- Preflight assessment and policy routing (`local_only`, `cloud_only`, `local_first`, `expert_first`, `adaptive`, `budget_constrained`).
- Budgets, escalation, and provider abstraction (Ollama, OpenAI-compatible, DeepSeek).
- Verification and context compression/handoff.
- Compute orchestration across local and cloud models.

## Must Not

- Hold project state or orchestrate work.
- Send data to the cloud without redaction.
- Hard-code model names in routing logic.
- Bypass approval or budget.

## Interfaces

- RPC `route(request) -> decision`.
- RPC `generate(messages, tier) -> result`.
- Progress stream for the control plane.

## Rules

1. Classification can raise, never lower, severity.
2. Major and critical decisions are forced through human approval.
3. Redaction runs before any cloud handoff and is covered by tests.
4. Budgets and token limits are enforced, not merely configured.
5. Routing logic is deterministic and brand-free; providers are configuration.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Contracts](../contracts/README.md)
