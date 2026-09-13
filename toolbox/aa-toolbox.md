# aa-toolbox.md

Agent directive for the `toolbox` module (aa-toolbox).

## Responsibility

Let the platform gain tools and change workflows without changing core code, while keeping capability grants explicit and enforcing sandbox rules.

## Owns

- Tool, plugin, and MCP manifests (identity, version, capabilities, permissions, I/O, sandbox needs).
- Capability and permission grants.
- Sandboxing and isolation.
- Provider-agnostic invocation RPC.
- Workflow definition schema, compiler, and runtime.
- The policy engine.

## Must Not

- Grant a capability implicitly or by default.
- Let a plugin reach outside its declared permissions.
- Require a core change to add a tool or MCP server.

## Interfaces

- Tool manifest and invocation RPC.
- Workflow runtime consumed by `kernel`.
- Permission decisions consumed by execution contracts.

## Rules

1. Declared capabilities intersect with the caller's execution contract before invocation.
2. Denials are explicit and audited.
3. Workflows are versioned, resumable, and testable as data.
4. MCP servers are ordinary providers behind the same permission model.

## Canonical references

- [Architecture — tooling and workflows](../docs/architecture/README.md#9-tooling-pluginsmcp-and-configurable-workflows)
- [Contracts](../contracts/README.md)
