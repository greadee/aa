# ADR-0133 — Rename console to ui with cli and tui surfaces

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-14) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The unified interface module was `aa-console` — a scaffolded module described as
"TUI and local web control plane, built on pluggable surfaces". The name
narrowed the whole interface to a console, and the module had no internal
structure.

## Decision

Rename the module **`console` → `ui`** (`github.com/greadee/aa/ui`) and give it a
structure: `components`, `views`, `state`, and `surfaces/`. Surfaces are `cli`
and `tui` for now; web and desktop surfaces are reserved. The directive
`console/aa-tui.md` becomes `ui/aa-ui.md`.

## Rationale

"ui" names the whole human interface rather than one surface, and the folder
structure makes the components/views/state/surfaces split explicit. It keeps the
pluggable-surface rule: adding a web or desktop surface must not change other
modules.

## Alternatives and consequences

- **Keep `console`:** rejected — the name is narrower than the module's role.
- **Add web/desktop surfaces now:** rejected — scope; only `cli` and `tui` are
  scaffolded in this refactor.
- Consequence: `go.work`, `tools/archtest`, CI, `MANIFEST.json`, and current-state
  docs were updated. Historical phase/ADR records keep the former name.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-14
- Module documentation: [docs/modules/ui](../modules/ui/README.md)
- [ADR index](./README.md)
