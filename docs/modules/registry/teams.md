# registry/teams

> Project history for the `registry/teams` submodule of [aa-registry](./README.md).

**Responsibility** — Durable team-definition store: a
grouping of related roles for organization, policy, and performance analysis.
Teams group roles; crews group workers at runtime. `TeamSpec` re-exports the
contracts v2 `team_spec`; `Registry` stores and serves validated definitions
(`Add` rejects duplicates; `Get`/`List`/`Count` read deterministically).

**Source** — [`registry/teams/`](../../../registry/teams/)

**Parent module** — [aa-registry](./README.md) · [docs/modules](../README.md)
