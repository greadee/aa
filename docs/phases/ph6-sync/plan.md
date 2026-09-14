# Phase 6 — Sync: Plan

> Authored at the **start** of the phase. Branch and folder: `ph6-sync`.

## Objective

Deliver `aa-sync`'s remote/offline/parallel capability: paired and mutually authenticated peers, a deterministic transport seam with an in-memory fake, resumable hash-verified chunked transfer, one-way revision sync with tombstones and deletion guards, an offline change queue with reconciliation, work-package and artifact distribution, multi-machine parallel coordination that carries no execution authority, and a transport-agnostic RPC adapter. The entire capability is testable offline.

## Starting State

- Starting ref: `main @ 83fe7a1` (after phase 5 merge and record).
- Available:
  - `aa-contracts` v1 schemas and Go bindings, the `sync.distribute` / `sync.fetchArtifact` / `sync.status` RPC method names, and the RPC envelope conventions (`contracts/rpc/rpc-v1.md`).
  - A scaffolded `sync/` Go module (`go.mod`, `doc.go`, `README.md`, `aa-sync.md`).
  - The architecture boundary harness allowing `sync -> contracts` (`tools/archtest`).
  - The phase documentation conventions and templates (`docs/github-os/phase-documentation.md`).
- Missing:
  - Any sync implementation; `sync` holds `doc.go` only.
  - Identity, pairing, and peer authentication.
  - A transport seam, a deterministic fake, and the transfer/chunk protocol.
  - Revision sync, the offline queue, and reconciliation.
  - Work-package/artifact distribution, parallel coordination, and the RPC adapter.
- Known constraints:
  - Sync must never become a remote shell and never gain or grant execution authority.
  - Authentication is identity-based; loopback and a pairing token are not authentication.
  - Receiver-authoritative apply; fail closed on validation.
  - Sync ordering is derived from record fields, never the wall clock.
  - Privacy defaults apply to everything that leaves a node.
  - The default test suite makes no network calls; live transport is a follow-up.
  - The phase branch name equals the phase folder name (`ph{N}-{scope}`) and the phase PR exists as a Draft from phase start.

## Scope

### In Scope
- Node identity (Ed25519 key pairs), deterministic test identities, pairing tokens, and mutual authentication.
- A `Transport` seam and a deterministic in-memory `fake` wire carrying envelope messages between named peers.
- Content-addressed manifests, chunking, per-chunk and whole-content hash verification, and resumable transfer.
- One-way revision sync: diff, apply, tombstones, and deletion guards; ordering by record revision.
- An offline change queue and reconciliation that classifies converged, to-send, to-apply, and conflicting changes.
- Work-package and artifact distribution with idempotent transfer IDs and receipts.
- Deterministic multi-machine coordination: balanced assignment of work packages to nodes, with no execution API.
- A transport-agnostic RPC adapter for `sync.distribute`, `sync.fetchArtifact`, and `sync.status`.

### Out of Scope
- Real sockets, mTLS, private tunnels, discovery, and relay selection; the fake and a transport interface stand in. A live transport is a follow-up.
- Execution or scheduling of the work it moves; sync coordinates transport only.
- Filesystem watching and a daemon; changes are supplied as records.
- Compression and encryption-at-rest; hashes detect corruption, not adversaries. A live transport owns confidentiality.

### Required End State
- [ ] Paired peers authenticate mutually; a wrong key or stale token fails closed.
- [ ] Chunked transfer resumes from a partial receiver and verifies every byte by hash.
- [ ] Revision sync applies one-way with tombstones and refuses unsafe deletions.
- [ ] The offline queue reconciles deterministically and converges.
- [ ] Work packages and artifacts distribute idempotently; repeated calls return the original receipt.
- [ ] Parallel coordination balances packages across nodes and exposes no execution surface.
- [ ] `sync.distribute`, `sync.fetchArtifact`, and `sync.status` dispatch and replay idempotently.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P6-001 | Identity is an Ed25519 key pair; the peer ID is the key fingerprint | Verifiable identity without a certificate authority; loopback is not authentication |
| ADR-P6-002 | Pairing uses a single-use, time-bounded token plus a mutual challenge-response | Pairing alone is not authentication; both sides prove possession of their key |
| ADR-P6-003 | The `Transport` seam is small (`Dial`/`Send`/`Receive`/`Close`) with a deterministic in-memory fake | Provider independence and offline end-to-end tests |
| ADR-P6-004 | Content is addressed by hash; every chunk and the whole payload are verified | Resumable transfer must fail closed on corruption |
| ADR-P6-005 | Transfer is resumable from a receiver's received-chunk set | Partial transfers and reconnects must not restart from zero |
| ADR-P6-006 | Revision sync is one-way and receiver-authoritative; deletion is a guarded tombstone | Prevents replica divergence and accidental data loss |
| ADR-P6-007 | Ordering derives from a record revision counter, never the wall clock | Deterministic convergence across clocks |
| ADR-P6-008 | Distribution is idempotent by a derived transfer key | Reconnects and retries must not duplicate transfers |
| ADR-P6-009 | Parallel coordination assigns work packages to nodes; it carries no execution authority | Sync moves work; it never starts an agent runtime |
| ADR-P6-010 | The RPC adapter is transport-agnostic and reuses the transport seam | The kernel can host it without coupling; tests stay offline |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase and its intended infrastructure. **Inputs** — architecture, contracts, the sync directive. **Expected Output** — `docs/phases/ph6-sync/plan.md` + `plan.uml`. **Model Class** — Strong. **Commit Message** — `add ph6 sync phase plan`. **Validation** — docs link check. **Dependencies** — none.

### Slice 2 — Core types, errors, and helpers
**Goal** — Domain types, sentinel errors, idempotency keys, and a deterministic clock. **Inputs** — the sync directive. **Expected Output** — `sync/types.go`, `sync/errors.go`, `sync/idempotency.go`, `sync/clock.go` + tests. **Model Class** — General. **Commit Message** — `add sync core types and errors`. **Validation** — `go test ./...`. **Dependencies** — slice 1. **Documentation** — `sync/README.md`.

### Slice 3 — Identity and pairing
**Goal** — Key-pair identity, peer fingerprints, pairing tokens, and mutual challenge-response authentication. **Inputs** — slice 2. **Expected Output** — `sync/identity/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add sync identity and pairing`. **Validation** — `go test ./...`; wrong key and stale token fail closed. **Dependencies** — slice 2. **Documentation** — `sync/README.md`.

### Slice 4 — Transport seam and deterministic fake
**Goal** — Envelope messages, the `Transport` interface, and an in-memory wire with per-peer queues, injection, and deterministic delivery. **Inputs** — slices 2–3. **Expected Output** — `sync/transport/*.go` + tests. **Model Class** — General. **Commit Message** — `add sync transport seam and fake`. **Validation** — `go test ./...`. **Dependencies** — slice 3. **Documentation** — `sync/README.md`.

### Slice 5 — Resumable chunked transfer
**Goal** — Content-addressed manifests, chunking, hash verification, and resume from a partial receiver. **Inputs** — slice 4. **Expected Output** — `sync/transfer/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add sync resumable transfer`. **Validation** — `go test ./...`; resume and corruption tests. **Dependencies** — slice 4. **Documentation** — `sync/README.md`.

### Slice 6 — Revision sync and offline queue
**Goal** — One-way revision diff/apply with tombstones and deletion guards, plus the offline queue and reconciliation. **Inputs** — slices 2, 4. **Expected Output** — `sync/revision/*.go`, `sync/queue/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add sync revision sync and offline queue`. **Validation** — `go test ./...`; convergence and guard tests. **Dependencies** — slice 5. **Documentation** — `sync/README.md`.

### Slice 7 — Distribution and parallel coordination
**Goal** — Work-package and artifact distribution with receipts and idempotency, and deterministic balanced assignment across nodes. **Inputs** — slices 5, 6. **Expected Output** — `sync/distribute/*.go`, `sync/parallel/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add sync distribution and parallel coordination`. **Validation** — `go test ./...`. **Dependencies** — slice 6. **Documentation** — `sync/README.md`.

### Slice 8 — RPC adapter
**Goal** — Transport-agnostic handlers for `sync.distribute`, `sync.fetchArtifact`, and `sync.status`. **Inputs** — `contracts/rpc/rpc-v1.md`. **Expected Output** — `sync/rpc/*.go` + tests. **Model Class** — General. **Commit Message** — `add sync rpc adapter`. **Validation** — `go test ./...`; idempotent replay. **Dependencies** — slice 7. **Documentation** — `sync/README.md`.

### Slice 9 — Phase summary and UML
**Goal** — Record the as-built phase. **Inputs** — all prior slices. **Expected Output** — `summary.md` + `summary.uml` + updated `sync/README.md`. **Model Class** — Strong. **Commit Message** — `add ph6 sync phase summary`. **Validation** — link check. **Dependencies** — slice 8.

## Exit Criteria

- [ ] Mutual authentication succeeds for paired peers and fails closed otherwise.
- [ ] A transfer resumes from a partial receiver and rejects corrupted chunks.
- [ ] Revision sync converges and honors deletion guards.
- [ ] The offline queue reconciles deterministically.
- [ ] Distribution is idempotent; no duplicate transfers.
- [ ] Parallel assignment is balanced and exposes no execution surface.
- [ ] `sync.*` methods dispatch and replay idempotently.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | identity fingerprints, pairing tokens, idempotency keys, chunk math, deletion guards |
| contract | `sync.*` RPC params/results; envelope compatibility |
| integration | transfer over the fake wire; revision convergence; queue reconciliation |
| security | wrong key/stale token fail closed; no credential fields; no execution API |
| determinism | stable transfer IDs, chunk order, assignment order, receipts |
| chaos | interrupted transfer resumes; duplicate distribution is a no-op |
| boundary | `sync` imports only contracts (archtest) |
