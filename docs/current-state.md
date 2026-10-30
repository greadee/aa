# Current state and limitations

> **Status:** v1 core composed; Phase 12 hardening and Phase 13 release work are pending.
> **Verified by:** `go run ./app selfcheck` and the per-module test suites.
> **Last reconciled:** Sprint 3 (2026-10-30).

This page is the canonical answer to "what works today?". Historical phase and
update records explain how the code got here and are not current-state claims.

## Delivered

- **Contracts v2** — JSON Schema sources, generated Go/TypeScript/Python
  bindings, RPC shapes, and conformance fixtures.
- **Control plane** — `app/` daemon exposing the OpenAPI HTTP/JSON API on
  loopback, with projects, work packages, graphs, approvals, assignments, nodes,
  sync status, tools, observations, visualizer, registry, and lifecycle reads.
- **Console** — CLI/TUI surfaces plus a read-only web surface, all through the
  control-plane client; a deterministic fake client supports offline use.
- **Kernel** — allocator/routing, planner, scheduler, gates, context compiler,
  observability, and job learning.
- **Runtime** — worker instances, sandbox isolation, and result intake. The
  optional Python inference sidecar is delivered as a service but is not wired
  into the default v1 daemon.
- **Memory** — canonical records, store, repositories, projection, retrieval,
  retention, and promotion.
- **Registry** — roles, models, teams, routines, policies, and capabilities as
  durable definitions.
- **Toolbox** — tool/plugin/MCP registry, policy engine, host, idempotent
  invocation, and workflow runtime.
- **Forge** — lifecycle, template, fake/real adapter, and memory-sync libraries.
  The v1 daemon constructs the memory-sync exporter but does not drive live
  GitHub mutations.
- **Sync** — identity, pairing, transfer, revision sync, distribution,
  parallel coordination, and the single-node registry used in v1.
- **Observation** — protocol, emitter, journal, transport, reports, and Codex
  translation.
- **Visualizer core** — adapter, source, replay, layout, live bridge, browsing,
  compatibility profile, and retention.

## Partial or limited

| Area | What exists | Limitation |
|---|---|---|
| Authentication | Loopback bind plus bearer-token carriage | Loopback is not authentication; real authn/authz (pairing, device keys, mTLS) is Phase 12 |
| Execution | Sandbox, worker, intake, and scheduler dispatch | Execution is disabled by default and must be enabled explicitly |
| Sync | Single-node fabric, nodes/status read model | Multi-machine execution authority and split-brain handling are v2 |
| Visualizer | Deterministic replay/layout/browse core | The Wails/React/Three 3D surface is deferred; core is data-only |
| Inference | Python sidecar and provider abstraction | Optional; the deterministic core path needs no model |
| Registry | Teams, routines, and policies stores | Default registries start empty; definitions are loaded per state root |
| Media | Media request in `docs/media/README.md` | Product screenshots, GIF/video, and the GitHub social-preview upload remain owner actions |
| Release | Version string and CI | No tags, installers, release workflow, or `CHANGELOG.md` until Phase 13 |

## Known limitations

- **Security:** no threat model, real authentication, authorization, or secret
  store yet. Treat the daemon as a local development surface only.
- **Packaging:** there is no installer, signed binary, container image, or
  supported release artifact.
- **Performance:** no load, soak, or chaos testing has been run.
- **Scale:** v1 is single-authority and single-node; replicas do not gain
  execution authority.
- **Visualization:** the 3D/Wails surface is not shipped; the shipped
  visualizer is the deterministic core and control-plane read model.
- **Docs media:** the owner must provide the product screenshots, GIF/video, and
  social-preview upload; none are committed yet.
- **Contracts:** binding parity compares field presence, optionality, and type
  categories in both directions. It is structural, not a full type-system
  equivalence proof; runtime fixture conformance covers values.

## Deferred to v2

- Desktop surface (Wails/React).
- Wails/React/Three visualizer surface.
- Multi-writer or multi-machine execution authority.
- Learned routing, embeddings, predictions, and crew recommendation until their
  evidence gates pass.

## How to verify

```sh
go run ./app selfcheck
cd tools && go run ./cmd/archtest -root .. && go test ./...
```

Per-module validation commands are in [../CONTRIBUTING.md](../CONTRIBUTING.md)
and each module README.

## Sources of truth

- [Architecture](architecture/README.md) — target design and decisions.
- [Module index](index/modules.md) — current module and submodule docs.
- [ADR index](adr/README.md) — accepted decisions.
- [Phase index](index/phases.md) and [update phases](updates/README.md) — historical work records.
- [Definitions of Done](github-os/definitions-of-done.md) — completion bar.
