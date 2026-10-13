# ISS-REVU Resolution Sprint — Plan

> **Type:** update phase (`iss-revu`). **Branch:** `iss-revu` → `dev`. **Source:** the accepted exceptions from `revu-findings` (PR #21 review).
> **Issues:** [ISS-REVU-1…#40](https://github.com/greadee/aa/issues/40), [#39](https://github.com/greadee/aa/issues/39), [#37](https://github.com/greadee/aa/issues/37), [#38](https://github.com/greadee/aa/issues/38).
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Close the four accepted exceptions recorded by the `revu-findings` sprint by implementing real
solutions rather than accepting them, so a `revu` audit of the branch reports zero findings — disjoint
from phase 10 (`ph10-console`/`ph10-ui`).

## Scope / decisions

- **ISS-REVU-1 — `computeruse.NewProvider` needs a real Driver.** Add a declared, non-test
  `computeruse.Headless` driver (`NewHeadless`) that fails closed with `toolbox.ErrUnsupported`, and
  wire `NewProvider(NewHeadless())` into the composition root (`app/`). No desktop backend required,
  so this is toolbox + app work, not UI. **Commit** `wire a declared headless computer-use driver`.
- **ISS-REVU-2 — `computeruse.NewFake` test double.** Remove the test double from the production
  export surface: unexport it (`fake`/`newFake`) and give the cross-module consumer (`kernel`) its own
  local `fakeDriver`. **Commit** `remove the computeruse test double from the export surface`.
- **ISS-REVU-3/4 — `tracesource.NewStatic`/`NewFake`.** These are used only by in-package tests;
  unexport them (`static`/`newStatic`, `fake`/`newFake`). **Commit** `unexport the tracesource test
  implementations`. (Folded into the same commit as ISS-REVU-2's test-surface cleanup when one message
  describes both.)

## Validation gates

| Gate | Command |
|---|---|
| Formatting | `gofmt -l .` |
| Go | per-module `go build/vet/test ./...` (`toolbox`, `kernel`, `app`) |
| Boundaries | `tools/archtest -root ..` |
| Review | `revu --profile pr` on the branch → 0 findings |

## Exit criteria

- [ ] `revu` reports zero findings for the branch (all four exceptions resolved).
- [ ] ISS-REVU-1…4 closed; the resolution PR closed/merged; PR #21 closed out.
