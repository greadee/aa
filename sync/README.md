# aa-sync

Remote sync, offline file sharing, and parallelization across machines.

- **Owns:** paired mutual-TLS transport; resumable verified chunked transfer; one-way revision sync; offline change queue and reconciliation; work-package and artifact distribution; parallelization transport.
- **Must not:** become a remote shell or gain execution authority.
- **Status:** delivered in `ph6-sync` (transport seam with a deterministic fake; a live transport is a follow-up).
- **Directive:** [aa-sync.md](aa-sync.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)

## Packages

| Package | Responsibility |
|---|---|
| `sync` | Domain types, sentinel errors, idempotency keys, and a deterministic clock |
| `identity` | Ed25519 identity, pairing tokens, and mutual challenge-response |
| `transport` | The transport seam and a deterministic in-memory wire |
| `transfer` | Content-addressed manifests, chunking, and resumable verified transfer |
| `revision` | One-way revision diff/apply with tombstones and deletion guards |
| `queue` | The offline change queue and deterministic reconciliation |
| `distribute` | Idempotent work-package and artifact distribution |
| `parallel` | Balanced, deterministic placement of work across nodes (no execution) |
| `rpc` | The `sync.distribute`, `sync.fetchArtifact`, and `sync.status` adapter |

## Testing

```sh
cd sync
go build ./... && go vet ./... && go test ./...
```

The default suites are offline and deterministic: they use the in-memory
`transport.Wire`, deterministic identities, and a fixed clock. No test makes a
network call. See [docs/phases/ph6-sync](../docs/phases/ph6-sync/plan.md) for
the phase plan and test plan.
