# visualizer — Module Update Summary: obsv transport adoption and AAV protocol de-duplication

> Authored at the **end** of the module update, on branch `ph8-visualizer`. Derived from [plan.md](plan.md).
> Phase PR: [#19](https://github.com/greadee/aa/pull/19)
> Sub-problems: `ISS-OBSV-2a` (transport adoption), `ISS-OBSV-2b` (protocol/IPC/event-store de-duplication) · Dependency: `ISS-OBSV-1` (`ph3-kernel`)

## Delivered

| Slice | Sub | Status | Commit | Notes |
|---|---|---|---|---|
| Issue documentation and module plan | — | Complete | `38b5669` | `docs/issues/ph8-visualizer-obsv-adoption.md`, this plan |
| Branch sync (prerequisite) | — | Complete | `a6a1146` | merged `main` to obtain the `ISS-OBSV-1` `obsv` API |
| obsv event adapter | 2a | Complete | `827720f` | `visualizer/adapter` (single `protocol.Event` → `v1.Event` mapping) |
| Remove browse fake event source | 2a | Complete | `832c5c4` | `browse_test` uses the real memory store; no fake providers |
| obsv event source | 2a | Complete | `ca2319b` | `source.OBsv`; `source.Fake` deleted |
| obsv metadata profile | 2a | Complete | `5377dc4` | `compat` v1.1 normalizes delimited real observation metadata |
| obsv live bridge | 2a | Complete | `676f5d0` | `bridge.Follow`; live equals replay over the real transport |
| obsv source config | 2a | Complete | `8b4e7d1` | `source.Config`/`Open`; in-process now, kernel socket later |
| Single protocol guard | 2b | Complete | `69d5f8a` | `archtest.CheckVisualizerBoundaries`, wired into `cmd/archtest` |
| Convergence documentation | 2b | Complete | `b71fa8f` | directive, README, architecture §3.1/§5.2/§5.9/§14 |
| Module update summary | — | Complete | this commit | this file |
| Phase PR link | — | Pending | — | follow-up slice |

## What was added

- `visualizer/adapter` — the one explicit mapping from the shared `obsv/protocol.Event` to the visualizer event type (`contracts` `v1.Event`): session, sequence, actor, work-package/attempt identity, and observation metadata; deterministic identifier normalization; a deterministic timestamp fallback. It is the only production package that imports `obsv/protocol`.
- `visualizer/source` — `OBsv` reads a session through `obsv` `Replay` and follows it through `obsv` `Subscribe`, both mapped by the adapter. `source.Fake` was deleted: there is no fake provider. `Config`/`Open` select the transport (in-process now; the kernel socket later) without changing callers. `ErrUnavailable` is the typed error for an unreachable transport.
- `visualizer/compat` — profile v1.1 normalizes the comma-separated scalar form the `obsv` durable allowlist preserves (`secondary_paths`, `access_sequence`) as well as the list form; unknown metadata is ignored.
- `visualizer/bridge` — `Follow` subscribes through the obsv-backed `source` and consumes into the same projection/layout path replay uses, so a followed session and its replay are the same frames.
- `tools/archtest` — `CheckVisualizerBoundaries` enforces the single observation vocabulary: only `adapter` maps `obsv/protocol`; only `source` consumes `obsv/transport`/`obsv/journal`; no package declares a second observation protocol, transport, or event-store. It runs as part of `go run ./cmd/archtest`.
- Documentation: the directive, README, and architecture §3.1/§5.2/§5.9/§14 now state that the shared `obsv` protocol and the `compat` profile replace AAV's duplicated protocol, IPC, and event-store.

## What this changes about the module and the app

- The visualizer now consumes the real `obsv` observation transport behind the existing `source.EventSource` seam; live and replay read the same events over the same transport and share one projection and render path.
- The AAV duplicate vocabulary is retired: `obsv` owns the observation protocol, transport, and event-store, and a boundary guard keeps it that way.
- The kernel-host socket remains the next transport step; `source.Open` already selects transports, so callers do not change when it lands.

## Is this part of a larger change?

No umbrella. This is the standalone issue `ISS-OBSV-2`; it depends on `ISS-OBSV-1` (`ph3-kernel`) for the `obsv` API and the durable allowlist keys, and it consumes them without editing `obsv`. The cross-process live path additionally waits on the kernel-host socket (`ISS-OBSV-1` follow-up).

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | all `visualizer/...` packages |
| Go tests | Pass | 81 tests across 10 packages |
| Formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `tools/archtest` ok: layering plus the visualizer single-protocol guard |
| Adapter | Pass | `TestToContractMapsFields`, `TestToContractOutputIsAValidContractEvent`, `TestToContractNormalizesIdentity`, `TestToContractRejectsInvalidObservation` |
| Source | Pass | `TestReplayReadsOBsvAndMaps`, `TestSubscribeReceivesAppended`, `TestReplayUnavailableTransport` |
| Profile over real metadata | Pass | `TestNormalizeDelimitedRealObservationMetadata`, `TestReplayMetadataFeedsCompatibilityProfile` (real `protocol.Sanitize` → transport → adapter → `compat`) |
| Live equals replay | Pass | `TestConsumeLiveEqualsReplay`, `TestFollowLiveEqualsReplay` (both read replay back through the same transport) |
| Source selection | Pass | `TestOpenDefaultsToInProcess`, `TestOpenExplicitTransport`, `TestOpenSocketUnavailable`, `TestOpenUnknownMode` |
| Single-protocol guard | Pass | rule tests for a second protocol import, a second transport import, a duplicate transport/event-store, and a duplicate observation type; `TestVisualizerBoundariesWorkspace` |
| No fakes / no kernel | Pass | no `fake`/`mock`/`stub` in `visualizer/*.go`; no `aa/kernel` import |
| Docs links | Pending | runs on CI after push |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P8U-001 | The visualizer consumes the `obsv` API behind the existing `source.EventSource` seam; the fake becomes test-only | Affirmed (stricter) | `source.OBsv`; `source.Fake` removed, not merely test-only |
| ADR-P8U-002 | One explicit adapter maps `obsv/protocol.Event` to the visualizer event type; the visualizer defines no protocol | Affirmed | `visualizer/adapter`; guard `single-obsv-adapter` |
| ADR-P8U-003 | `visualizer/compat` remains the single metadata normalizer; the profile keys are representable in the `obsv` allowlist | Affirmed | `compat` v1.1; `obsv` allowlist keys; `TestReplayMetadataFeedsCompatibilityProfile` |
| ADR-P8U-004 | The wire socket stays the kernel host's; the visualizer reaches it without changing callers | Affirmed | `source.Open`; `ModeSocket` reports `ErrUnavailable` until the socket lands; guard `single-obsv-transport` |
| ADR-P8U-005 | Live and replay keep one projection and one render path over the real transport | Affirmed | `bridge.Follow`; `TestFollowLiveEqualsReplay` |
| ADR-P8U-006 | De-duplication is enforced by a boundary test, not convention alone | Affirmed | `archtest.CheckVisualizerBoundaries`, four rules |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | The `ISS-OBSV-1` `obsv` API is available on the branch | The branch was based on `main @ d9f8477`, before PR #15; `obsv` was scaffold-only | Merged `origin/main` (`a6a1146`) to obtain the dependency before slice 2 |
| 2 | Slice 3: `source.Fake` becomes test-only | All fake providers removed (including `browse`'s) and tests use the real in-memory `obsv` transport and real memory store | No fake provider should remain in the visualizer test suite (directed) |
| 3 | Slice 5 delivers the live bridge over the real transport | Bridge tests moved to the real transport in slice 3 (the fake was deleted); slice 5 added `Bridge.Follow` and a streaming live-equals-replay test | Removing the fake in slice 3 forced the bridge tests to the real transport |
| 4 | `compat` normalizes `secondary_paths`/`access_sequence` over real metadata | Added delimited-scalar normalization (profile v1.1); the `obsv` allowlist keeps only scalar values | The real transport preserves the keys as comma-separated strings (per `obsv/protocol` tests) |
| 5 | A kernel-host socket transport | `ModeSocket` is represented in `Config` and reports a typed `ErrUnavailable`; the socket client is owned by `ISS-OBSV-1` | The socket is a kernel-host concern; the selection seam is in place so callers do not change |

## Follow-Up

- Kernel-host obsv socket; `source.Config.Transport` accepts the client when it lands (under `ISS-OBSV-1`).
- Kernel orchestrator wiring: an `obsv`-backed `orchestrator.Sink`.
- Durable in-process journal wiring for the visualizer (a file-backed `transport.NewJournal`).
- The Wails + React/Three UI surface over `replay.Frame` (phase 8 follow-up).

## Metrics

- Commits: 1 plan + 1 branch-sync merge + 8 implementation/closeout + this summary
- Packages: 10 (`visualizer`, `adapter`, `source`, `compat`, `session`, `layout`, `replay`, `bridge`, `browse`, `retention`); `tools/archtest` extended
- Tests: 81 visualizer tests + 13 `tools` tests
- ADRs: ADR-P8U-001 … ADR-P8U-006
- Sub-problems completed: `ISS-OBSV-2a`, `ISS-OBSV-2b`
- PRs: 1 (module update, `ph8-visualizer` → `main`)

## Retrospective

### Repeat
- Route every observation through one adapter at one seam; "live equals replay" reduced to comparing the frames of a followed subscription with a replay read back through the same transport.
- Enforce boundaries with a static guard, not convention; the four rules caught the exact defect (a second protocol/transport/store) the architecture flagged.
- Normalize the real wire form (scalar, delimited) in the profile rather than widening `obsv`; the allowlist stays scalar and metadata-only.

### Avoid
- Do not let a module update start on a branch that predates its declared dependency; verify the dependency is present before planning slices.
- Do not keep a stand-in transport provider once the real transport exists; deleting the fake (rather than keeping it test-only) kept the tests honest.

### As-Built Diagram

The visualizer phase diagram in [../docs/phases/ph8-visualizer/summary.uml](../../../../phases/ph8-visualizer/summary.uml) remains accurate; this update replaces the `source.Fake` stand-in with the `obsv` transport and adds the single-protocol guard.
