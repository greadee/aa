# runtime/inference

> Project history for the `runtime/inference` submodule of [aa-runtime](./README.md).

**Responsibility** — The model provider/execution service, renamed from the
Python `aa-sifter` module and reduced to provider/execution only. Its routing,
budget, verification, and context responsibilities are reserved for later
issues. The Python subproject will live here; the Go runtime invokes it over the
inter-module RPC boundary, never by import.

**Source** — [`runtime/inference/`](../../../runtime/inference/)

**Parent module** — [aa-runtime](./README.md) · [docs/modules](../README.md)
