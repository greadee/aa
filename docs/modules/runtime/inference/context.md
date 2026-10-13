# inference/src/aa_inference/context

> Project history for the Python `context` submodule of [aa-runtime](../README.md).

**Responsibility** — Execution-local provider-message fitting, the outbound secret-redaction chokepoint, local-task handoff, and the escalation packet.

**Source** — [`inference/src/aa_inference/context/`](../../../../runtime/inference/src/aa_inference/context/)

**Parent module** — [aa-runtime](../README.md) · [docs/modules](../../README.md)

## Boundary

This package is **execution-local**, not control-plane context:

- `compression.py` fits an already-decided provider message list to the target
  model's window (a runtime/providers concern).
- `handoff.py` / `packet.py` redact secrets before cloud egress and format an
  escalation prompt ([ADR-0050](../../../adr/ADR-0050-one-outbound-redaction-chokepoint-in-call-tier-for-every-expert.md)).

Control-plane context — needs discovery, selection, budgeting, provenance, and
assembly over retrieval candidates — moved to [`kernel/context`](../../kernel/context.md)
in the ten-issue sprint (Issue 3 / A8). See
[ADR-0146](../../../adr/ADR-0146-context-compilation-and-a8-extraction.md).
