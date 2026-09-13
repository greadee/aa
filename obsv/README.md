# aa-obsv

Product-neutral work observation: protocol, durable emitter, local transport, journal, and deterministic work reports.

- **Owns:** observation protocol v1, `Durable` projection, bounded failure-open emitter, attach-or-own local transport, journal with replay/subscribe/retention, work reports, Codex `exec` translator.
- **Must not:** call models, own graph state or canonical project history, or require a runtime.
- **Status:** scaffolded. Implementation migrates from the existing `aa-work-observation` source in `ph3-kernel`.
- **Directive:** [aa-obsrv.md](aa-obsrv.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
