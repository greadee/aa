# aa-toolbox

The tool, plugin, and MCP registry plus the configurable workflow runtime.

- **Owns:** tool/plugin/MCP manifests; capability and permission grants; sandboxing; provider-agnostic invocation RPC; workflow definition schema, compiler, and runtime; policy engine.
- **Must not:** grant capabilities implicitly.
- **Status:** delivered in `ph7-toolbox` (provider seam and deterministic fake; a live MCP host and an OS sandbox enforcer are follow-ups).
- **Directive:** [aa-toolbox.md](aa-toolbox.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)

## Packages

| Package | Responsibility |
|---|---|
| `toolbox` | Re-exported contracts, domain types, sentinel errors, idempotency keys, and a deterministic clock |
| `registry` | Validated tool/plugin/MCP manifests and the `Provider` seam |
| `policy` | Capability intersection with the execution contract, sandbox decisions, and the ordered audit log |
| `host` | Provider-agnostic invocation with idempotent replay, plus a deterministic fake |
| `workflow` | The workflow compiler (validate + deterministic topological plan) and the resumable runtime |
| `rpc` | The `toolbox.invoke`, `toolbox.compileWorkflow`, and `toolbox.registry` adapter |

## Testing

```sh
cd toolbox
go build ./... && go vet ./... && go test ./...
```

The default suites are offline and deterministic: they use the in-memory fake
provider, a fixed clock, and a map-backed contract resolver. No test makes a
network call. See [docs/phases/ph7-toolbox](../docs/phases/ph7-toolbox/plan.md)
for the phase plan and test plan.
