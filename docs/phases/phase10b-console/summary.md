# Phase 10b — Console — Summary

> End-of-phase summary. Derived from [plan.md](./plan.md) and [design.md](./design.md).
> Branch: `phase10b-console`.

## Delivered

- **Control-plane client** (`ui/controlplane`): a typed HTTP/JSON client over
  `contracts/openapi/control-plane-v1.yaml`, a deterministic in-memory `Fake`, and governed-error
  decoding. The console never imports module internals.
- **Read models + views** (`ui/state`, `ui/views`): the view-model layer (status, projects, work,
  assignments, nodes, tools, active view) and a deterministic, metadata-only text renderer
  (dashboard: nav column + main pane + status bar) with a plain-text mode.
- **Commands + audit** (`ui/command`): palette parsing (`open`, `submit`, `approve`,
  `create-project`, `refresh`, `help`, `quit`) and an idempotent, keyed audit log.
- **Text surface** (`ui/surfaces/tui`): the console dashboard with single-key navigation
  (`j`/`k`, `[`/`]`), palette commands, and a line-oriented `Run` loop.
- **Web surface** (`ui/surfaces/web`): a read-only inspector over the same client — semantic HTML
  plus a `/plain` text alternative.
- **Entry point**: `aa console [--web] [--control --token --project --addr]` wires a surface to the
  client (text surface by default; `--web` serves the inspector).

## Commits

| Commit | Change |
|---|---|
| `7b9e7d4` | add console UI/TUI design decisions |
| `6d7abe3` | add control-plane client and fake |
| `ead6b15` | add console read models and views |
| `4af7122` | add console commands and text surface |
| `6ca0c0c` | add console web surface |
| _(this)_ | add console entry point, tests and phase summary |

## Validation

| Gate | Result |
|---|---|
| `gofmt` / `go build` / `go vet` / `go test` (`ui`, `app`) | pass |
| `tools/archtest -root ..` | pass (`ui` imports only `contracts`; `app` is the composition root) |
| Integration smoke | `aa console` (text + `--web`) driven against the phase-10a control plane: create project → submit work → browse projects/nodes/tools; web `/` and `/plain` render |

## Deviations / notes

- The text surface is a **line-oriented dashboard** in v1 (stdlib only): keybindings via single-key
  input or `:command` lines. A full-screen raw-terminal TUI with cursor navigation is a follow-up.
- The web surface is **read-only** (browse only) in v1, per the design decision.
- The `reports` view is a placeholder pending a phase-9 learning read model on the control plane.
- Approvals navigate the single-project assignment list; richer gate UX is future work.

## Exit criteria

- [x] A single entry point drives the control plane through a chosen surface; no module imports.
- [x] Text + web surfaces share one view-model layer; deterministic, golden-testable output.
- [x] Typed commands are validated, idempotent, audited.
- [x] Tests pass against the fake + `httptest`.
