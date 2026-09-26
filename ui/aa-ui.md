# aa-ui.md

Agent directive for the `ui` module (aa-ui).

## Responsibility

Provide one place for a human to drive and inspect the whole platform, without coupling to module internals.

## Owns

- The `cli` and `tui` surfaces.
- (Reserved) the local web and desktop surfaces.
- Workflow configuration.
- Plugin and MCP management.
- Memory, issue, strategy, and visualizer browsing.
- Multi-machine status.
- Approval and gate interactions.

## Must Not

- Import module internals; only `contracts` and the control-plane API.
- Reimplement orchestration or memory logic.
- Make one surface a prerequisite for another.

## Interfaces

- Control-plane API client (versioned, authenticated).
- A **surface** interface implemented by the CLI, the TUI, and any future web or desktop surface.

## Rules

1. Surfaces are pluggable; adding a web or desktop surface must not change other modules.
2. Every write goes through the control-plane API and is idempotent and audited.
3. The UI works offline against the local host.
4. Approvals and gates are always explicit about consequences.

## Canonical references

- [Architecture — ui](../docs/architecture/README.md#510-aa-ui)
- [Contracts](../contracts/README.md)
