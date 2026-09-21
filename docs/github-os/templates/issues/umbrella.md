# ISS-<AREA>-<slug> — <problem>

**Type:** technical debt / investigation / feature
**Status:** open (umbrella)
**Affects branches:**

## Problem

## Scope boundary

This umbrella records the problem, the distinct sub-problems, and the branch dependency order. It intentionally does not specify solutions for branches whose work has not started.

## Sub-problems

These are deliberately separate problems and are not one issue.

| Id | Problem | Module / branch | Why it matters | Status | Document |
|---|---|---|---|---|---|
| ISS-<AREA>-1 | ... | `<module>` / `ph{N}-{scope}` | ... | planned | `docs/issues/<branch>-<slug>.md` |

## Dependency order

```text
ISS-<AREA>-1
    -> ISS-<AREA>-2
    -> ...
```

## Ownership

No phase is added for this work. Each sub-problem is completed on the module phase that already owns it, with the phase's retained branch reopened for a module update.
