# ADR-0132 — Move runtime out of kernel to a top-level runtime module

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-2, D-9, D-10) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

Execution mechanics lived inside `aa-kernel` as `kernel/runtime` (the
provider-neutral worker adapter plus a deterministic fake and a disabled
adapter). The target architecture separates *coordination* — planning,
allocation, and scheduling — from *execution mechanics*, and reserves a home for
sandboxing and for the model provider/execution service.

## Decision

Move `kernel/runtime` to a top-level **`runtime/`** module. The moved adapter
becomes `runtime/worker`; `runtime/lifecycle` and `runtime/sandbox` are reserved
boundaries (sandboxing is **not** implemented), and `runtime/inference` is the
placeholder for the Python provider/execution service renamed from `sifter`.
The kernel depends on `runtime` and dispatches allocated work to it.

## Rationale

Clear separation of concerns: the kernel decides *what/whose/which/how much*;
runtime decides *how it runs*. It reserves the sandbox integration point without
implementing it, and gives the renamed inference service a home inside the
execution boundary.

## Alternatives and consequences

- **Keep runtime inside the kernel:** rejected — execution mechanics are not
  coordination and would keep two jobs in one module.
- **Put the Python inference service at top level:** rejected — it belongs to the
  execution boundary and is nested under `runtime/`.
- Consequence: `kernel` now imports `aa-runtime`; layering and `tools/archtest`
  were updated. Sandbox behavior is deferred to the ten-issue phase.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-2, D-9, D-10
- Module documentation: [docs/modules/runtime](../modules/runtime/README.md)
- [ADR index](./README.md)
