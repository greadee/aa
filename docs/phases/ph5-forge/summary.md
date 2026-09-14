# Phase 5 — Forge: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `99987b8` | plan.md + plan.uml |
| Core types and interfaces | Complete | `adc275e` | `forge/{types,validate,interfaces,idempotency,audit,clock}.go` |
| Deterministic fake: repos and issues | Complete | `9bba8f3` | `forge/fake` (idempotency, audit log, injection) |
| Fake PRs, checkpoints, releases | Complete | `c026841` | human-gated merge; Draft PRs |
| Audits and findings | Complete | `2cb1ec0` | severity ordering; derived result |
| Template engine | Complete | `643e7a1` | renders `docs/github-os/templates` |
| Memory synchronization | Complete | `ae61613` | `forge/memsync` -> contracts |
| Phase lifecycle orchestrator | Complete | `5fbe93a` | `forge/lifecycle` e2e over the fake |
| RPC adapter | Complete | `d5f49a9` | `forge/rpc` (`createIssue`, `openPullRequest`, `checkpoint`, `release`) |
| Phase summary and UML | Complete | this commit | summary.md + summary.uml + README |

- Starting ref: `main @ 41de261` (after phase 4 merge and record).
- Tracking issue: none (phase executed via the phase PR).
- Milestone: none (matches phases 0–4 on this repository).
- Phase PR: [#6](https://github.com/greadee/aa1/pull/6)
- Branch: `ph5-forge` (retained after merge on `aa1`)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | all `forge/...` packages |
| Go tests | Pass | 46 tests across 6 packages |
| Formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `archtest` reports ok; `forge` imports only contracts |
| Full phase lifecycle | Pass | `StartPhase` -> child issue -> child PR -> checkpoint -> audit -> release -> gated merge, over the fake |
| Idempotency | Pass | repeated repository/issue/branch/PR/checkpoint/release calls return the original object |
| Human gate | Pass | merging without an approving actor returns `ErrHumanGate` |
| Contract mapping | Pass | `v1.Issue` and `v1.MemoryRecord` validate |
| Determinism | Pass | stable IDs, audit order, and rendered templates |
| RPC | Pass | `forge.*` methods dispatch; idempotent replay; unknown method rejected |
| Docs links | Pass | runs on CI after push |
| Phase audit | Complete | see Audit below |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P5-001 | Forge abstraction with capability sub-interfaces and a deterministic fake | Affirmed | the whole lifecycle runs against `fake` with no network |
| ADR-P5-002 | Every mutating operation is idempotent | Affirmed | replay tests for repository, branch, issue, PR, checkpoint, release |
| ADR-P5-003 | Merging is a distinct, human-gated operation | Affirmed | `MergePullRequest` fails closed without an actor |
| ADR-P5-004 | Templates are data from `docs/github-os/templates` | Affirmed | `template.Engine` loads and renders the versioned files |
| ADR-P5-005 | Branch name equals the phase folder name; phase PR starts Draft | Affirmed | `StartPhase` validates `ph{N}-{scope}` and opens a Draft PR |
| ADR-P5-006 | Forge objects map to contracts via a `Sink` | Affirmed | `memsync` maps and validates; the fake sink records |
| ADR-P5-007 | The fake is the e2e substrate | Affirmed | deterministic runs and stable audit order |
| ADR-P5-008 | Audits classify and do not auto-fix | Affirmed | `Audit` stores findings; `FindingsFromAudit` yields issue specs |
| ADR-P5-009 | The forge never stores secrets | Affirmed | no credential fields exist in any type |
| ADR-P5-010 | RPC adapter is transport-agnostic and idempotent | Affirmed | `rpc.Service.Handle` takes/returns envelope objects; replay cache |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | GitHub and local git adapters | Deferred; the `Forge` interface plus the deterministic fake are delivered | CI has no credentials and must make no network calls; adapters are a follow-up |
| 2 | Branch objects as real git branches | Branches recorded as forge objects | Real git object manipulation needs a workspace manager (kernel); the forge owns the lifecycle record |
| 3 | Terse RPC params from the spec sketch | Pragmatic params (`repoId` + a structured object) | The spec lists method names, not exact shapes; the adapter documents and tests the chosen shapes |
| 4 | Process templates as human skeletons | Added Go template placeholders to the github templates | The forge renders them as data; the skeletons remain readable |
| 5 | Merge gate as a GitHub review requirement | Merge gate as an explicit `MergeApproval` actor checked by the forge | The review mechanism is provider-specific; the explicit gate is portable and fail-closed |
| 6 | Audit result always supplied by the caller | Derived from severities when omitted (P0/P1 block, P2 conditions, else ready) | Deterministic, matches `repository-audits.md` |
| 7 | Immediate phase merge | `CompletePhase` checkpoints, marks the PR ready, then merges | Mirrors the real lifecycle and keeps the audit trail complete |

## Deferred / Follow-Up

- A `gh`-backed `Forge` adapter and a local-git adapter, tested with a command-runner fake.
- Hosting `forge/rpc` over the local socket and wiring the kernel's forge client.
- A memory-backed `Sink` that writes issue and checkpoint records to `aa-memory` over RPC.
- Filing audit findings as issues automatically (specs exist; the loop is a console/phase-next concern).
- Template authoring guidance: the versioned templates now carry Go placeholders.

## Metrics

- Commits: 9 implementation + this summary
- Packages: 6 (`forge`, `fake`, `template`, `memsync`, `lifecycle`, `rpc`)
- Tests: 46
- ADRs: ADR-P5-001 … ADR-P5-010
- Pitfalls recorded: 0
- PRs: 1 phase PR (#6)
- Issues: none

## Audit

| Severity | Count | Notes |
|---|---|---|
| P0 | 0 | — |
| P1 | 0 | — |
| P2 | 0 | — |
| P3 | 2 | Live adapters and the memory writer (documented above) |

Audit result: no blocking findings. Boundary rule holds: `forge` imports only `contracts` (plus its own subpackages).

## Retrospective

### Repeat
- Build the fake first and drive the end-to-end lifecycle through it; it forced the interfaces to be minimal and testable.
- Idempotency keys plus replay assertions caught duplicates before any provider existed.
- Parameterize the process templates early so rendering and docs share one source.

### Avoid
- Keep capability interfaces narrow; the composite `Forge` only belongs in `lifecycle`, where every capability is needed.
- Derive deterministic defaults (audit result, IDs, timestamps) instead of leaving them to the caller.
- Do not expose merge over RPC; keep the human gate on the control-plane path.

### As-Built Diagram

[summary.uml](./summary.uml)
