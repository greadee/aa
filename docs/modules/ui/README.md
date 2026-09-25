# aa-ui — module documentation

> Project history for the `aa-ui` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-ui.md`](../../../ui/aa-ui.md) · **Source** — [`ui/`](../../../ui/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Provide one place for a human to drive and inspect the whole platform, without
coupling to module internals.

## Owns

- The `cli` and `tui` surfaces.
- (Reserved) the local web and desktop surfaces.
- Workflow configuration.
- Plugin and MCP management.
- Memory, issue, strategy, and visualizer browsing.
- Multi-machine status.
- Approval and gate interactions.

## Must not

- Import module internals; only `contracts` and the control-plane API.
- Reimplement orchestration or memory logic.
- Make one surface a prerequisite for another.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`components`](./components.md) | Shared interface components |
| [`views`](./views.md) | View definitions rendered by surfaces |
| [`state`](./state.md) | Interface state (navigation, selection, connection) |
| [`surfaces/cli`](./surfaces-cli.md) | Command-line surface |
| [`surfaces/tui`](./surfaces-tui.md) | Terminal-UI surface |
