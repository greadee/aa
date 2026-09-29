# aa-ui

The unified interface: the CLI and TUI surfaces over the control plane, built on
pluggable surfaces so a web or desktop surface can be added later.

- **Owns:** the CLI and TUI surfaces; (reserved) local web control plane; workflow configuration; plugin/MCP management; memory/issue/strategy/visualizer browsing; multi-machine status; approval and gate UI.
- **Must not:** couple to module internals; it uses only the control-plane API and `contracts`.
- **Status:** scaffolded. Delivered last, in `ph10-ui`. Surfaces are `cli` and `tui` for now.
- **Directive:** [aa-ui.md](aa-ui.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)

## Packages

| Package | Responsibility |
|---|---|
| `ui` | Module root: the unified interface |
| `components` | Shared interface components |
| `views` | View definitions rendered by surfaces |
| `state` | Interface state |
| `surfaces/cli` | Command-line surface |
| `surfaces/tui` | Terminal-UI surface |
