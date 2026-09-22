# obsv — Module Update Summary: observation substrate

> Authored at the **end** of the module update, on branch `ph3-kernel`. Derived from [module-upd-plan.md](module-upd-plan.md).
> Phase PR: [#15](https://github.com/greadee/aa/pull/15)
> Sub-problem: `ISS-OBSV-1` · Parent umbrella: `ISS-TRACE-LOOP`

## Delivered

| Slice | Status | Commit message | Notes |
|---|---|---|---|
| Issue documentation and module plan | Complete | `add ph3 kernel obsv module update plan and issue documentation` | `docs/issues/`, this plan |
| Protocol v1 | Complete | `add obsv observation protocol` | `obsv/protocol` |
| Bounded emitter | Complete | `add obsv bounded emitter` | `obsv/emit` |
| Journal | Complete | `add obsv journal` | `obsv/journal` |
| Local transport | Complete | `add obsv local transport` | `obsv/transport` |
| Work reports | Complete | `add obsv work reports` | `obsv/report` |
| Codex translator | Complete | `add obsv codex exec translator` | `obsv/codex` |
| Runtime facade | Complete | `wire obsv runtime ensure send replay subscribe` | `obsv` root, README |
| Module update summary | Complete | `add ph3 kernel obsv module update summary` | this file |
| Phase PR link | Complete | `link ph3 kernel obsv pull request` | follow-up commit |

## What was added

- `obsv/protocol` — observation protocol v1: source types (`session`, `tool`, `file`, `work_delta`), confidence (`exact`, `correlated`, `observed`, `inferred`), validation, and an allowlist sanitizer that drops content, prompt, source, and credential keys and bounds values.
- `obsv/emit` — a failure-open emitter that never blocks, propagates, or panics; it sanitizes and validates before writing and accounts for accepted/dropped/failed events, plus a bounded `Buffer` target.
- `obsv/journal` — an append-only, sequence-assigning journal with cursor `Replay`, live failure-open `Subscribe`, and a bounded projected `Trim`; in-memory and durable JSONL file backends that preserve sequence across reopen.
- `obsv/transport` — an in-process local transport with `hello`/`append`/`replay`/`subscribe` and attach-or-own session semantics, plus a session-bound `Client`.
- `obsv/report` — deterministic work reports (counts by source type and confidence, sorted work packages and actors) independent of input order.
- `obsv/codex` — a tolerant Codex `exec` JSONL translator with deterministic sequence assignment.
- The `obsv` facade: `Ensure`/`Runtime`, `Send`, `Replay`, `Subscribe`, `Report`, and a `Manager` that owns one journal per session.

## Post-review extension — visualizer profile metadata keys

After this summary was authored, `ISS-OBSV-2` (visualizer transport adoption, `ph8-visualizer`) identified that the `visualizer/compat` profile keys `secondary_paths` and `access_sequence` were not in the durable allowlist and so could not survive sanitization over the shared transport. They are observation metadata, not content, so they were added to `obsv/protocol`'s allowlist (`add obsv visualizer profile metadata keys`) so the profile can normalize them. No behavior change beyond preserving these two allowlisted keys; the deny list is unchanged.

## What this changes about the module and the app

- `aa-obsv` is no longer a scaffold: it now produces the observation evidence that the trace contract and trace store consume. It remains product-neutral and imports no other aa module.
- The kernel owns the obsv host per the architecture; the kernel can now open a session with `obsv.Ensure`, stream events with `Runtime.Send`, and read them back with `Replay`/`Subscribe`.
- No other module or contract changes; `contracts` continues to defer observation events to `aa-obsv`.

## Is this part of a larger change?

Yes. This is the observation sub-problem (`ISS-OBSV-1`) of the umbrella change `ISS-TRACE-LOOP` (the observation → trace → learning substrate was under-delivered; phase 3 deviation #2). It depends only on the substrate level of `ISS-TRACE-1` and `ISS-TRACE-2` and unblocks live capture on the kernel host and the learning sub-issue `ISS-LEARN-1`. Those solutions are not specified here.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet / test | Pass | 7 packages, 48 test functions under `obsv/` |
| Formatting | Pass | `gofmt -l obsv` clean |
| Durability | Pass | `TestFileJournalSurvivesReopen`, `TestDurableJournalSurvivesReopen` |
| Determinism | Pass | report input-order independence; translator repeat-stability |
| Failure-open | Pass | emitter isolates a panicking and an erroring sink, never propagates |
| Boundaries | Pass | `tools/archtest` reports ok; `obsv` imports no other aa module |
| All modules | Pass | `go build/vet/test` across every Go module |
| Docs links | Pending | runs on CI after push |

## Decisions affirmed

| # | Decision | Outcome |
|---|---|---|
| ADR-P3U-001 | `obsv` owns observation protocol v1 | Affirmed — no `contracts` change required |
| ADR-P3U-002 | Metadata-only allowlist sanitization | Affirmed — denied keys dropped at `Send` and in the translator |
| ADR-P3U-003 | Bounded, failure-open emitter | Affirmed — panics and errors are counted, never propagated |
| ADR-P3U-004 | Append-only, replay-equivalent journal; retention is a projected view | Affirmed — `Trim` does not mutate the journal |
| ADR-P3U-005 | Attach-or-own local transport | Affirmed — a second greet/dial attaches to the existing owner |
| ADR-P3U-006 | Sequence-based ordering, never wall clock | Affirmed — the journal assigns the sequence |
| ADR-P3U-007 | Confidence is never promoted | Affirmed — `Promotable()` is always false |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | A wire-level local socket | An in-process `transport.Local` with the same `hello`/`append`/`replay`/`subscribe` surface | The socket transport belongs with the kernel host; the in-process transport fixes the protocol shape and is testable without a runtime |
| 2 | Retention bounds the journal | Retention returns a projected view | Honours ADR-P3U-004 and the `visualizer/retention` precedent; the canonical record is `memory` |

## Follow-Up

- Kernel host wiring: an `obsv`-backed `orchestrator.Sink` so the control plane records live observation (under `ISS-OBSV-1`).
- A wire transport (local socket) for the kernel host; the visualizer's adoption and the AAV protocol de-duplication are tracked by `ISS-OBSV-2` (`ph8-visualizer`).
- Trace capture that promotes observation into the `contracts` `trace` object for the `ph2-memory` store.
