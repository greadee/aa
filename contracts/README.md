# aa-contracts

The single source of truth for every cross-module object and wire protocol.

- **Owns:** JSON Schemas (event, task/work-package/DAG, execution contract, result envelope, telemetry, project records, memory objects, issue/strategy, route request/response, tool/MCP manifest, workflow definition), the RPC/transport spec, generated Go/TypeScript/Python bindings, the conformance suite, and versioning policy.
- **Must not:** contain business logic.
- **Status:** scaffolded. Delivered in `ph1-contracts`.
- **Directive:** [aa-contracts.md](aa-contracts.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
