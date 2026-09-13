# aa-console

The unified interface: TUI and local web control plane, built on pluggable surfaces.

- **Owns:** CLI/TUI and local web control plane; workflow configuration; plugin/MCP management; memory/issue/strategy/visualizer browsing; multi-machine status; approval and gate UI.
- **Must not:** couple to module internals; it uses only the control-plane API and `contracts`.
- **Status:** scaffolded. Delivered last, in `ph10-console`.
- **Directive:** [aa-tui.md](aa-tui.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
