# v1 — Toolbox Pre-Hardening Milestone

## Milestone Description (to be set on GitHub)

```
Pre-hardening sprint for the toolbox module (ph7-toolbox dependency).

Addresses all findings from the toolbox module code review before ph7-toolbox is merged to main.

## Issues

### P0 — Critical
- [ ] #??? — RPC TOCTOU race condition in Service.Handle (concurrent duplicate execution)

### P1 — High
- [ ] #??? — Stale contract version in test fixtures (1.0 vs 2.0)
- [ ] #??? — RPC layer uses map[string]any (no compile-time safety)

### P2 — Medium
- [ ] #??? — No timeout honored in RPC handler (TimeoutMS ignored)
- [ ] #??? — RPC error responses lack structured data
- [ ] #??? — Untyped WorkflowStep.Config (no validation)
- [ ] #??? — No metrics or observability

### P3 — Low
- [ ] #??? — Large contract files should be split
- [ ] #??? — Python/TypeScript binding parity review

## Sprint Progress

| Slice | Issue | Status |
|---|---|---|
| Fix RPC TOCTOU race | #??? | Planned |
| Update test fixtures to v2 | #??? | Planned |
| Add typed RPC request structs | #??? | Planned |
| Honor TimeoutMS in RPC | #??? | Planned |
| Add structured RPC error data | #??? | Planned |
| Validate WorkflowStep.Config | #??? | Planned |
| Add metrics interface | #??? | Planned |
| Split large contract files | #??? | Planned |
| Review Python/TypeScript bindings | #??? | Planned |

## Sprint Exit Criteria

- [ ] all issues complete
- [ ] no open P0/P1 bugs
- [ ] toolbox test suite passes
- [ ] contracts conformance suite passes
- [ ] ready for main merge
```

## Sprint Plan

See [../updates/toolbox-pre-hardening/plan.md](../updates/toolbox-pre-hardening/plan.md).

## PR

See [../updates/toolbox-pre-hardening/pr-body.md](../updates/toolbox-pre-hardening/pr-body.md).
