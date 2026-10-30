# aa-ui

The unified interface: the CLI and TUI surfaces over the control plane, built on
pluggable surfaces so a web or desktop surface can be added later.

- **Owns:** the CLI, TUI, and read-only local web surfaces; workflow configuration; plugin/MCP management; memory/issue/strategy/visualizer browsing; multi-machine status; approval and gate UI.
- **Must not:** couple to module internals; it uses only the control-plane API and `contracts`.
- **Status:** CLI, TUI, and read-only local web surfaces delivered over the control-plane client, including the offline fake client. The desktop surface is deferred to v2.
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
