# Future UI / TUI / GUI surfaces

> **Placeholder.** Concrete surfaces beyond the terminal are not yet planned. This folder is
> reserved for the UI/TUI/GUI work that follows Phase 10, Phase 10a, and Phase 10b, and this
> PR is intentionally left open as the running home for that work.

## Context

Phase 10 (`ph10-console`) delivers the interface core: the real control-plane server and the host composition root. It is split into parts:

- **Phase 10a — Terminal:** the thin console client and its deterministic text/terminal surface.
- **Phase 10b — Surface:** the pluggable `Surface` interface and the shared view-model layer.

See `docs/phases/ph10-console/plan.md` and the part plans under `docs/phases/ph10-console/parts/` on the `ph10-console` branch (Phase 10 PR #11).

## Scope of this placeholder

The following surfaces are **out of scope** for Phase 10/10a/10b and will be planned and delivered here, after those merge:

- local web surface;
- desktop (Wails/React/Three) surface;
- any additional TUI/GUI surface.

Each is a concrete implementation of the `Surface` interface from Phase 10b. They add their own
dependencies on this branch without widening any module boundary.

## Status

No plans or code yet. This document is a placeholder; the branch and PR are deliberately left
open for future UI/TUI/GUI work once Phase 10, 10a, and 10b have landed.
