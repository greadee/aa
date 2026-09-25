# aa-contracts — module documentation

> Project history for the `aa-contracts` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-contracts.md`](../../../contracts/aa-contracts.md) · **Source** — [`contracts/`](../../../contracts/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Define every object and protocol that crosses a module boundary, exactly once, and generate bindings from it.

## Owns

- JSON Schemas for all cross-module objects.
- The RPC/transport specification.
- Generated Go, TypeScript, and Python bindings.
- The conformance suite.
- Schema versioning and compatibility policy.
- The control-plane OpenAPI document.

## Must not

- Contain business logic.
- Depend on any other module.
- Allow modules to hand-roll cross-module DTOs.

## Interfaces

- Generated packages consumed by every module.
- Schemas consumed by tooling and tests.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`go/v1`](./go-v1.md) | Package v1 contains the Go bindings for the aa contract schemas v1. |
| [`openapi`](./openapi.md) | Control-plane OpenAPI specification. |
| [`python`](./python.md) | Hand-authored Python binding (`aa_contracts`) and its tests. |
| [`rpc`](./rpc.md) | Inter-module JSON-RPC v1 specification and control-plane OpenAPI. |
| [`schemas/v1`](./schemas-v1.md) | JSON Schema 2020-12 sources of truth for every cross-module object. |
| [`typescript`](./typescript.md) | Hand-authored TypeScript binding and typecheck. |

