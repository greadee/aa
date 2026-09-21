# Issues

Durable issue documentation. GitHub issues are the execution record; these documents are the knowledge record for work that must survive a session.

## Convention

- One document per issue or sub-issue. Do not collapse distinct problems into a single document.
- A larger change that spans several modules and branches gets an **umbrella** document that lists its distinct sub-issues and the branches they affect. The umbrella states the problem and the dependency order; it does not specify the solution for a branch whose work has not started.
- Each sub-issue's solution plan is authored with the pull request on the branch that owns it, and is linked from that sub-issue's document.
- Module-level plans and summaries live with the module they change (`<module>/module-upd-plan.md`, `<module>/module-upd-summary.md`) and are linked from the owning sub-issue document.

## Index

| Document | Issue | Scope | Branches |
|---|---|---|---|
| [trace-learning-substrate.md](trace-learning-substrate.md) | ISS-TRACE-LOOP | Umbrella: the observation → trace → learning substrate is under-delivered | `ph1-contracts`, `ph2-memory`, `ph3-kernel`, `ph4-sifter`, `ph9-joblearn` |
| [ph1-contracts-trace-contract.md](ph1-contracts-trace-contract.md) | ISS-TRACE-1 | `contracts`: bounded per-step trace object | `ph1-contracts` |
