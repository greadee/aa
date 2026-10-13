# Phase 10b — Console — UI/TUI Design

> Maintainer decisions for the console surfaces. **Kept out of `plan.md` on purpose**; this is the
> design record the 10b.4/10b.5 commits implement.

## Decisions

| Aspect | Decision |
|---|---|
| TUI toolkit | **Go standard library only** — raw terminal + ANSI; no TUI framework dependency |
| Layout | **Dashboard**: persistent left nav (views) + main pane + status/help bar |
| Interaction | **Command palette + keybindings** (`:`-style palette, context shortcuts) |
| Visual style | **Minimal / accessible**: monochrome by default, no color dependency, screen-reader friendly, plain-text alternative |
| Web surface | **Read-only inspector first** (browse read models); commands later |
| Components (priority order) | 1 Projects & work · 2 Graph & assignments · 3 Issues/strategies/memory · 4 Nodes & tools · 5 Learning reports · 6 Approvals & gates |

## Layout

```
┌ aa ────────────────────────────────────────────────────────────────┐
│ nav                 │ main pane                                     │
│ ▸ Projects & work   │  <view model rendered here>                   │
│ ▸ Graph             │                                               │
│ ▸ Issues/Memory     │                                               │
│ ▸ Nodes & tools     │                                               │
│ ▸ Learning reports  │                                               │
│ ▸ Approvals & gates │                                               │
├─────────────────────┴───────────────────────────────────────────────┤
│ : palette   ? help   ·   connected <host>   ·   node <id>   ·   …    │
└─────────────────────────────────────────────────────────────────────┘
```

Single alternate-screen; minimum 80×24; degrades to a plain-text, line-oriented mode when the
terminal is not interactive (pipes/CI) so output stays golden-testable.

## Interaction

- `:` — command palette (`open <view>`, `submit work`, `approve`/`reject`, `config get|set`, `refresh`, `quit`).
- `?` — contextual help; `Tab`/`Shift-Tab` (or `[`/`]`) — cycle nav; arrows / `j`/`k` — move;
  `Enter` — select/open; `/` — filter; `R` — refresh; `q`/`Ctrl-C` — quit.
- Approvals always require an explicit confirm (never auto-resolved).

## Surface / view-model contract

- One **view-model layer** shared by the text surface and the local web surface; rendering is
  `viewModel → string` (text) or `viewModel → HTML` (web). No view logic in the client.
- Rendering is **metadata-only**: never print secrets, raw prompts, or source.

## Accessibility

- No meaning carried by color alone; Unicode box-drawing with an ASCII fallback.
- Plain-text alternative for every web view.
