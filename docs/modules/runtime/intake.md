# runtime/intake

> Project history for the `runtime/intake` submodule of [aa-runtime](./README.md).

**Responsibility** — Package intake validates untrusted result envelopes and deduplicates them. A result is data until the authority accepts it; intake never grants authority, it only normalizes, validates, and detects conflicts.

**Source** — [`runtime/intake/`](../../../runtime/intake/)

**Parent module** — [aa-runtime](./README.md) · [docs/modules](../README.md)

## Boundary

Result intake is execution mechanics, so it lives in the `runtime` module: it
validates the output of a worker attempt before the control plane accepts it.
`kernel/scheduler` consumes it across the module boundary (kernel → runtime).
Moved from `kernel/intake` in the ten-issue sprint (Issue 8 / A5), per
[ADR-0143](../../adr/ADR-0143-decide-intake-and-contract-ownership.md).
