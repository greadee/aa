# visualizer — Module Update Plan: obsv transport adoption and AAV protocol de-duplication

> Authored at the **start** of the module update, on branch `ph8-visualizer` (the phase that owns `aa-visualizer`).
> Standalone issue `ISS-OBSV-2` · sub-problems `ISS-OBSV-2a` (transport adoption) and `ISS-OBSV-2b` (de-duplication) · dependency `ISS-OBSV-1` (`ph3-kernel`).

## Objective

Replace the visualizer's stand-in `EventSource` fake with the real `obsv` `Replay`/`Subscribe` transport behind the existing seam, map `obsv` observation events into the visualizer's `contracts` event type through one adapter, apply the explicit compatibility profile to real observation metadata, and retire the duplicated AAV observation protocol/IPC/event-store so `obsv` is the only observation vocabulary.

## Starting State

- Starting ref: `main @ d9f8477`; branch `ph8-visualizer` fast-forwarded to it.
- Available:
  - The phase 8 visualizer packages: `source` (`EventSource` seam + `Fake`), `compat` (versioned profile for `secondary_paths`/`access_sequence`), `session`, `layout`, `replay`, `bridge`, `browse`, `retention`, and core types (`visualizer.Event = contracts` `v1.Event`).
  - The `contracts` event taxonomy (`contracts/go/v1/event.go`).
  - The `obsv` module API delivered by `ISS-OBSV-1` (`ph3-kernel`, PR #15): `protocol.Event`, `Replay`, `Subscribe`, `Report`, and the attach-or-own in-process transport.
  - The architecture boundary allowing `visualizer -> contracts, obsv, memory` (`tools/archtest`).
  - The memory query seam for past sessions (`browse`).
- Missing:
  - Any transport binding in the visualizer: the real `obsv` `Replay`/`Subscribe` is not consumed.
  - An adapter from `obsv/protocol.Event` to the visualizer event type.
  - Proof that the `compat` profile works over real observation metadata.
  - The single-protocol guard that retires AAV's duplicate protocol/IPC/event-store.
  - The kernel-host wire socket (owned by `ISS-OBSV-1`).
- Known constraints:
  - `obsv` is the only observation protocol; the visualizer must not define one.
  - The `obsv` durable allowlist must include `secondary_paths`/`access_sequence` before the profile can read them; the extension is requested under `ISS-OBSV-1`.
  - Ordering derives from `Sequence`, never the wall clock; live and replay share one path.
  - Observation metadata is metadata-only; no file contents or raw prompts.
  - `visualizer` never imports `kernel`; the wire socket is reached over RPC.
  - The default test suite makes no network calls and is deterministic.

## Scope

### In Scope

- An `obsv`-backed `EventSource` (`Replay`/`Subscribe`); the fake becomes test-only.
- One explicit `obsv/protocol.Event` → `contracts` `v1.Event` adapter.
- The `compat` profile exercised over real observation metadata.
- Live bridge parity over the real transport.
- Source selection between the in-process transport and the future socket.
- The single observation protocol/transport/event-store boundary guard (2b).
- The AAV convergence recorded in the directive, README, and architecture.
- The module update plan and summary; the module update pull request.

### Out of Scope

- The kernel-host wire socket and the kernel `obsv`-backed sink (`ISS-OBSV-1`).
- Editing the `obsv` module or its allowlist (requested under `ISS-OBSV-1`).
- The Wails + React/Three UI surface and camera controls.
- Canonical history and promotion (owned by `memory`).

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P8U-001 | The visualizer consumes the `obsv` module API behind the existing `source.EventSource` seam; the fake becomes test-only | One seam already exists; swapping the implementation must not change projection, layout, or replay |
| ADR-P8U-002 | One explicit adapter maps `obsv/protocol.Event` to the visualizer event type; the visualizer defines no protocol | Keeps one vocabulary and one mapping point; the visualizer stays a consumer |
| ADR-P8U-003 | `visualizer/compat` remains the single metadata normalizer; the profile keys must be representable in the `obsv` allowlist | Adopting observation metadata stays deliberate and tested; the allowlist extension is requested under `ISS-OBSV-1` |
| ADR-P8U-004 | The wire socket stays the kernel host's; the visualizer reaches it without changing callers | Transport ownership is `obsv`/kernel; the visualizer adds no transport of its own |
| ADR-P8U-005 | Live and replay keep one projection and one render path over the real transport | Preserves the phase 8 invariant; prevents live/replay drift |
| ADR-P8U-006 | De-duplication is enforced by a boundary test, not convention alone | A second protocol/transport/event-store is the original defect; the guard keeps it retired |

## Slices

Each slice maps to exactly one commit and is authored after this plan is reviewed. Slice 1 is delivered by the pull request carrying this plan.

| Slice | Sub | Goal | Commit message |
|---|---|---|---|
| 1 | — | Issue documentation and this plan | `add ph8 visualizer obsv adoption plan and issue documentation` |
| 2 | 2a | `obsv/protocol.Event` → `contracts.Event` adapter | `add visualizer obsv event adapter` |
| 3 | 2a | `obsv`-backed `EventSource`; fake → test-only | `add visualizer obsv event source` |
| 4 | 2a | `compat` over real observation metadata | `add visualizer obsv metadata profile` |
| 5 | 2a | Live bridge through the real transport; live equals replay | `add visualizer obsv live bridge` |
| 6 | 2a | Source selection (in-process now, socket later) | `add visualizer obsv source config` |
| 7 | 2b | Single observation protocol/transport/event-store guard | `add visualizer single protocol guard` |
| 8 | 2b | AAV convergence in the directive, README, and architecture | `document visualizer obsv convergence` |
| 9 | — | Module update summary | `add ph8 visualizer obsv adoption summary` |
| 10 | — | Link the module update PR | `link ph8 visualizer obsv adoption pull request` |

## Required End State

- [ ] `visualizer/source` reads a session through `obsv` `Replay` and follows it through `obsv` `Subscribe`; the fake is test-only.
- [ ] One adapter maps `obsv/protocol.Event` to `contracts` `v1.Event`; no second protocol type exists in `visualizer`.
- [ ] `visualizer/compat` normalizes `secondary_paths`/`access_sequence` from real observation metadata; unknown metadata is ignored.
- [ ] Live output equals replay output over the real transport.
- [ ] No package under `visualizer/` declares an observation protocol, transport, or event-store; the boundary test enforces it.
- [ ] The default suite makes no network calls; `visualizer` never imports `kernel`.
- [ ] The directive, README, and architecture reflect the adopted state.
- [ ] A module update PR is opened on `ph8-visualizer` into `main`.

## Exit Criteria

- [ ] The visualizer live and replay paths consume `obsv` through one adapter and one seam.
- [ ] The compatibility profile is proven over real observation metadata.
- [ ] The single-protocol guard passes, and `tools/archtest` reports ok.
- [ ] `go build`, `go vet`, `go test`, and `gofmt` pass across the module.
- [ ] A module update PR is opened on `ph8-visualizer` into `main`.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | adapter field mapping, compat normalization from `obsv` metadata, source selection |
| contract | `obsv/protocol.Event` → `v1.Event` adapter output validates against `contracts` |
| integration | `obsv` journal → `Replay`/`Subscribe` → projection → layout → replay; live equals replay |
| boundary | no second protocol/transport/event-store in `visualizer`; `obsv` is the only observation source; `kernel` not imported (`tools/archtest`) |
| determinism | the same journal yields stable graph, layout, and frames; delivery order independent |
| failure | an unavailable `obsv` transport yields a typed error, not a panic |
| docs | every relative link resolves; docs link check |

## Dependencies And Follow-Up

- **`ISS-OBSV-1` (`ph3-kernel`)**: the `obsv` API and the durable metadata allowlist extension for `secondary_paths`/`access_sequence`. The visualizer does not edit `obsv`.
- **Kernel-host wire socket** (`ISS-OBSV-1` follow-up): needed for the cross-process live path; the in-process transport suffices until then.
- **UI surface**: the Wails + React/Three renderer remains a separate follow-up.
