# Modules

Project history: the current state of each aa module. Work history lives under [docs/phases](../phases/) (numbered major additions) and [docs/updates](../updates/) (unnumbered update phases).

Each module folder carries `README.md` (overview, owns/must-not, submodules) and one page per submodule, plus `updates/<slug>/{plan.md,summary.md}` for module updates. Module directives (`aa-<module>.md`) stay beside the module.

| Module | Responsibility |
|---|---|
| [aa-contracts](./contracts/README.md) | Define every object and protocol that crosses a module boundary, exactly once, and generate bindings from it. |
| [aa-registry](./registry/README.md) | The durable definition store: roles, models, teams, capabilities, routines, policies. Definitions only, no runtime instances. |
| [aa-obsv](./obsv/README.md) | Observe work events from agents, tools, files, and git, with bounded, failure-open delivery and deterministic replay. |
| [aa-runtime](./runtime/README.md) | Execution mechanics for allocated work: worker runtime, reserved lifecycle/sandbox, and the inference service. |
| [aa-kernel](./kernel/README.md) | Own deterministic coordination of all work: plan it, select roles and workers, dispatch it, gate it, and observe it. The kernel is the platform; models are workers. |
| [aa-memory](./memory/README.md) | Be the canonical, portable, rebuildable system of record. Everything durable about work, projects, issues, and strategies lives here. The SQLite database is a projection; the records are the truth. |
| [aa-sync](./sync/README.md) | Move files and work between machines securely and resiliently, and coordinate parallel work across machines without assuming execution authority. |
| [aa-sifter](./sifter/README.md) | Given a prompt and context, decide which model tier and provider should handle it, enforce budget and approval, execute, verify, and escalate on failure. The sifter is the sole path to any model. |
| [aa-forge](./forge/README.md) | Automate the Git and GitHub engineering lifecycle as first-class, auditable objects, and keep the durable record in `memory` in sync. |
| [aa-toolbox](./toolbox/README.md) | Let the platform gain tools and change workflows without changing core code, while keeping capability grants explicit and enforcing sandbox rules. |
| [aa-visualizer](./visualizer/README.md) | Make work tangible: render the current session live and any past session as a replayable, debuggable 3D graph. |
| [aa-ui](./ui/README.md) | Provide one place for a human to drive and inspect the whole platform, without coupling to module internals. |
| [github-os](./github-os/README.md) | Process documentation: issues/sub-problems, module updates, phases, sprints, PRs, review, audits, ADRs. |
