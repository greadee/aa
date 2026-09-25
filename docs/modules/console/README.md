# aa-console — module documentation

> Project history for the `aa-console` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-tui.md`](../../../console/aa-tui.md) · **Source** — [`console/`](../../../console/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Provide one place for a human to drive and inspect the whole platform, without coupling to module internals.

## Owns

- The CLI and TUI.
- The local web control plane.
- Workflow configuration.
- Plugin and MCP management.
- Memory, issue, strategy, and visualizer browsing.
- Multi-machine status.
- Approval and gate interactions.

## Must not

- Import module internals; only `contracts` and the control-plane API.
- Reimplement orchestration or memory logic.
- Make the desktop surface a prerequisite for the TUI/web surfaces.

## Interfaces

- Control-plane API client (versioned, authenticated).
- A **surface** interface implemented by the TUI, the web control plane, and any future desktop surface.

## Submodules

_No submodules yet (scaffold)._

