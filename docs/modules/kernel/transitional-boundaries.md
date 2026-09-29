# Kernel transitional boundaries and intended ownership

> Current placement is **not** final architectural ownership. This page records the
> intended home of each kernel package so the refactor does not entrench a
> transitional placement. See [architecture-refactor-1](../../updates/architecture-refactor-1/plan.md)
> decision D-15 and [ADR-0136](../../adr/ADR-0136-kernel-transitional-package-placement.md).

The architecture refactor moved planning, allocation, and scheduling into
`kernel/allocator` and `kernel/scheduler`, and moved execution mechanics and
worker instances to `aa-runtime`. The packages below **stay in the kernel for
now**; their intended ownership may differ and is recorded here.

| Package | Current | Intended | Status | Constraint |
|---|---|---|---|---|
| `context` | `kernel/context` | `kernel/context` — internal split into needs / selection / budgeting / compression / assembly / provenance reserved | stays | must not depend on `scheduler` or `runtime`; must not select roles, models, or compute |
| `contract` | `kernel/contract` | shared authority/budget builder consumed by the allocator (budget) and runtime (authority); may move under `allocator` or become a shared kernel package | transitional | must remain a leaf and must not import `scheduler` or `runtime` |
| `gate` | `kernel/gate` | `kernel/gate` — acceptance is control-plane policy | stays | leaf |
| `telemetry` | `kernel/telemetry` | `kernel/telemetry` — operational telemetry (decision D8) | stays | leaf |
| `joblearn` | `kernel/joblearn` | `kernel/joblearn` — the learning engine; apprenticing/studying vocabulary and attribution dimensions reserved | stays | reaches `memory` only through seams (`promote`, `tracesource`) |
| `api` | `kernel/api` | `kernel/api` — the control-plane surface; transport (HTTP/socket) lands with `ui` | stays | depends only on `scheduler` |
| `intake` | `kernel/intake` | **`runtime`** — result intake is execution mechanics | transitional | must not accrue new kernel dependents; today only `scheduler` uses it, keep it a seam |

## Rules

1. Do not treat the current placement of a **transitional** package as final.
2. Do not add dependencies that would entrench a transitional placement (no new
   package may import `intake` except the scheduler; `contract` stays a leaf).
3. Prefer the narrowest seam when a transitional package is consumed, so it can
   move without touching callers.
4. Document the intended home before moving anything.
5. A move is its own slice, with the module docs and an ADR updated.
