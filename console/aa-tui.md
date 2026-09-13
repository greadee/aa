# aa-tui.md

Agent directive for the `console` module (aa-console / TUI).

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

## Must Not

- Import module internals; only `contracts` and the control-plane API.
- Reimplement orchestration or memory logic.
- Make the desktop surface a prerequisite for the TUI/web surfaces.

## Interfaces

- Control-plane API client (versioned, authenticated).
- A **surface** interface implemented by the TUI, the web control plane, and any future desktop surface.

## Rules

1. Surfaces are pluggable; adding a desktop app must not change other modules.
2. Every write goes through the control-plane API and is idempotent and audited.
3. The console works offline against the local host.
4. Approvals and gates are always explicit about consequences.

## Canonical references

- [Architecture — console](../docs/architecture/README.md#510-aa-console)
- [Contracts](../contracts/README.md)
