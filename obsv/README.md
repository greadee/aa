# aa-obsv

Product-neutral work observation: protocol, durable emitter, local transport, journal, and deterministic work reports.

- **Owns:** observation protocol v1, `Durable` projection, bounded failure-open emitter, attach-or-own local transport, journal with replay/subscribe/retention, work reports, Codex `exec` translator.
- **Must not:** call models, own graph state or canonical project history, or require a runtime.
- **Status:** implemented. The kernel host wiring (an `obsv`-backed sink) is a follow-up.
- **Directive:** [aa-obsrv.md](aa-obsrv.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
- **Module update:** [module-upd-plan.md](module-upd-plan.md) · [module-upd-summary.md](module-upd-summary.md)

## Layout

| Package | Responsibility |
|---|---|
| `obsv` | facade: `Ensure`/`Runtime`, `Send`, `Replay`, `Subscribe`, reports |
| `obsv/protocol` | observation protocol v1, confidence, metadata allowlist |
| `obsv/emit` | bounded, failure-open emitter and buffer |
| `obsv/journal` | in-memory and durable JSONL journal with replay, subscribe, retention |
| `obsv/transport` | local attach-or-own transport (`hello`/`append`/`replay`/`subscribe`) |
| `obsv/report` | deterministic work reports |
| `obsv/codex` | Codex `exec` JSONL translator |
