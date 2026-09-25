# aa-toolbox — module documentation

> Project history for the `aa-toolbox` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-toolbox.md`](../../../toolbox/aa-toolbox.md) · **Source** — [`toolbox/`](../../../toolbox/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Let the platform gain tools and change workflows without changing core code, while keeping capability grants explicit and enforcing sandbox rules.

## Owns

- Tool, plugin, and MCP manifests (identity, version, capabilities, permissions, I/O, sandbox needs).
- Capability and permission grants.
- Sandboxing and isolation.
- Provider-agnostic invocation RPC.
- Workflow definition schema, compiler, and runtime.
- The policy engine.

## Must not

- Grant a capability implicitly or by default.
- Let a plugin reach outside its declared permissions.
- Require a core change to add a tool or MCP server.

## Interfaces

- Tool manifest and invocation RPC.
- Workflow runtime consumed by `kernel`.
- Permission decisions consumed by execution contracts.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`host`](./host.md) | Package host is the provider-agnostic tool invocation host. |
| [`policy`](./policy.md) | Package policy decides whether a caller may invoke a tool. |
| [`registry`](./registry.md) | Package registry holds validated tool, plugin, and MCP manifests and the provider that executes each tool kind. |
| [`rpc`](./rpc.md) | Package rpc exposes aa-toolbox over the aa inter-module RPC v1 methods. |
| [`workflow`](./workflow.md) | Package workflow validates declarative workflows into deterministic plans and runs them resumably. |

