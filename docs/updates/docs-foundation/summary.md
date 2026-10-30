# v1 docs foundation — summary

> **Status:** complete. **Branch:** `docs/v1-foundation` → `dev`. **PR:** #93.
> **Plan:** [plan.md](plan.md). **Diagram:** [summary.uml](summary.uml).

## What changed

The repository now has a current-state entry point, a runnable README, current
architecture diagrams, repository media, finished issue/PR definitions of done,
templates, a license, contributing guidance, reconciled status claims, and a
field-level cross-language contract parity gate.

### Delivered by issue

| Issue | Commit | Outcome |
|---|---|---|
| #85 `ISS-DOCS-1` | `4c5567d` | Final DoD, issue/PR/comment templates, `CONTRIBUTING.md` |
| #86 `ISS-DOCS-2` | `12dc282` | MIT license, repository description/homepage/topics, social-preview source |
| #88 `ISS-DOCS-4` | `b85c6bb` | Docs entry, audience paths, canonical-vs-historical map, current-state/limitations |
| #87 `ISS-DOCS-3` | `bb85dff` | README rewrite: product, status, quickstart, module map, media, links |
| #89 `ISS-DOCS-5` | `e35bb89` | Current composition, contracts, data-flow, deployment, and E2E diagrams |
| #90 `ISS-DOCS-6` | `c2b6805` | Terminal/console/web/architecture media and validation |
| #91 `ISS-DOCS-7` | `ccc5f74` | Stale-claim reconciliation, dated supersession notes, CLI reference |
| #76 `ISS-CONTRACTS-1` | `b019256` | `go/ast` field/optionality/type-category parity gate, drift fixes, self-tests, CI |

The docs entry/current-state slice (#88) landed before the README commit (#87) so
every README link was valid at the time it was committed. That ordering is the
only deviation from the plan's slice order.

## Decisions and deviations

- **MIT license** and repository metadata were recorded as planned.
- **Generated media** (terminal, console, web content, architecture) is committed;
  owner-branded video/screenshot and the GitHub social-preview upload remain
  explicit owner actions because GitHub has no API for the social preview.
- **Historical records preserve their original text.** `v1-readiness` D4 and the
  composition table have dated supersession notes instead of rewrites.
- **Go AST helper.** The parity gate uses `contracts/tools/goastdump` so grouped
  declarations are parsed correctly; CI's `contracts-python` job now sets up Go.
- **Optionality is defined by `omitempty`.** A pointer without `omitempty` is a
  required key that may encode as null, not an optional key. This made the
  `TaskGraph` required `projectId`/`revision` explicit.
- **Parity is structural.** The gate compares type categories and optionality,
  not full type-system equivalence; runtime fixtures still own value validation.
- **CI now runs on `dev` PRs**, so the parity gate and Go/TS suites cover the
  branch's actual merge path.

## Post-docs audit

An independent fresh-context audit reviewed the README, docs entry/current state,
indexes, architecture diagrams, templates, media, CLI reference, CI, and the
contract parity gate.

### Coverage ledger

| Area | Reviewed | Findings | Disposition |
|---|---|---|---|
| README / product accuracy | README, `app/main.go`, control-plane routes, composition imports | 2 | fixed |
| Docs entry / state / indexes | `docs/README.md`, `docs/index/*`, `current-state.md`, `v1-readiness` | 4 | fixed |
| Architecture / diagrams / modules | diagrams, architecture doc, UI/visualizer docs and `doc.go` files | 3 | fixed |
| Reference / CLI / manifest | CLI reference, reference index, `MANIFEST.json`, `go.work` | 1 | fixed |
| Process / templates / DoD | DoD, issue templates, PR template, comment templates | 1 | fixed |
| Media | media manifest, SVGs, README media section, media checker | 1 | fixed |
| Contract parity | gate, AST helper, self-tests, bindings | 2 | fixed |
| CI / validation | CI and docs-link workflows, docs checkers | 3 | fixed / documented |

### Findings disposition

| Finding | Severity | Disposition |
|---|---|---|
| F1 docs checkers untracked while workflow referenced them | P1 | fixed in the closeout commit |
| F2 CI did not run on PRs targeting `dev` | P1 | fixed: `dev` added to CI and docs-link triggers |
| F3 stale scaffolded/reserved claims in canonical code docs | P2 | fixed in `contracts`, `runtime`, `registry`, and module index |
| F4 forge/inference shown as default daemon wiring | P2 | fixed: marked as libraries/optional in README, current state, and diagrams |
| F5 Phase 13 referenced without a roadmap entry | P2 | fixed: Phase 13 row and program note added |
| F6 auth claim ignored empty-token behavior | P2 | fixed: docs now say the check is disabled when the token is empty |
| F7 target architecture classified as today's authority | P2 | fixed: as-built canonical and target design are separated |
| F8 parity gate latent gaps | P3 | fixed: TS/Python-only types and `Required` inside `total=False` are covered |
| F9 media command/alt-text inaccuracies | P3 | fixed: `python3` and an explicit alt-text column |
| F10 external links not actually checked | P3 | documented: offline lychee/checker policy is explicit |
| F11 trailing blank lines in comment templates | P3 | fixed |

A second fresh-context verification pass confirmed F1–F11 and raised four minor
follow-ups: a residual "reserved lifecycle" phrase (N1), a CONTRIBUTING external-
link contradiction (N2), a media table separator (N3), a mixed canonical heading
(N4), Phase 12 naming (N5), and the local PyYAML requirement (N6). All six were
fixed in the audit-follow-up commit.

**Sign-off:** no P0/P1 findings remain.

## Validation

| Command | Result |
|---|---|
| `python3 tools/docs/check_links.py` | local links and anchors ok |
| `python3 tools/docs/check_media.py` | media validation ok |
| `python3 tools/docs/check_templates.py` | issue and PR templates ok |
| `cd contracts && gofmt -l . && go build ./... && go vet ./... && go test ./...` | pass |
| `cd app && gofmt -l . && go build ./... && go vet ./... && go test ./...` | pass |
| `python3 -m unittest discover -s contracts/tools -p 'test_*.py'` | 11 tests, OK |
| `python3 -m unittest discover -s contracts/python/tests -t contracts/python` | 4 tests, OK |
| `python3 contracts/tools/check_binding_parity.py` | parity ok |
| `cd contracts/typescript && npx tsc --noEmit` | pass |
| Clean clone: `go run ./app selfcheck` | `selfcheck ok` |
| Clean clone: `aa serve` + `/api/v1/health` | 200 |
| Clean clone: `aa console --fake` | renders and exits cleanly |
| `cd tools && go run ./cmd/archtest -root ..` | boundaries ok |

## Residual risk

- Owner actions: social-preview upload and optional video/browser screenshot.
- External links are not checked while lychee runs with `--offline`; add a
  scheduled online run if external link rot becomes a concern.
- The parity gate is structural by design; runtime fixture conformance remains
  the value-level authority.
- Phase 12 hardening/security and Phase 13 release remain future work.
