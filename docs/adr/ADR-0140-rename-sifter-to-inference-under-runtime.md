# ADR-0140 — Rename the Python `sifter` service to `inference` under `runtime`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-1, D-2) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The Python `aa-sifter` module owned model routing, budgets, approval,
verification, and context compression, and lived beside `kernel`. The target
architecture places model execution under the runtime boundary and reserves the
routing/budget/verification/context responsibilities for later issues.

## Decision

Rename the Python module **`sifter` → `inference`**, move it to
**`runtime/inference`**, and reduce it to a model provider/execution service.
Rename the package `aa_sifter` → `aa_inference`, the distribution
`aa-sifter` → `aa-inference`, the CLI `aa-sifter` → `aa-inference`, the
environment prefix `SIFTER_` → `INFERENCE_`, the data dir `~/.aa_sifter` →
`~/.aa_inference`, and the RPC service `sifter.*` → `inference.*`. Identifiers
are renamed (`ComputeSifter` → `ComputeInference`, `SifterClient` →
`InferenceClient`, ...). Behavior is unchanged; routing/budget/verification/
context extraction is reserved.

## Rationale

The service belongs to the execution boundary, and "inference" names its
reduced role. It removes the last product-brand name from the module graph and
keeps the kernel's RPC reach consistent.

## Alternatives and consequences

- **Keep `sifter` at top level:** rejected — it is execution, not control, and
  the name implies the removed routing role.
- **Reimplement the split now:** rejected — that is new behavior; reserved.
- Consequence: the CI jobs, RPC spec, kernel `InferencePredictor` seam, module
  docs, and the `runtime/inference` subproject are renamed.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-1, D-2
- [ADR-0132 — Runtime module](./ADR-0132-move-runtime-out-of-kernel-to-a-top-level-runtime-module.md)
- [ADR index](./README.md)
