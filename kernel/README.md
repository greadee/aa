# aa-kernel

The deterministic control plane: planning, roles, orchestration, execution, gates, and job learning.

- **Owns:** obsv host; planning and DAG readiness; role/trade/worker registry and role selection; scheduler and assignment state machine; context compiler; execution contracts and budgets; runtime adapters; workspace/worktrees; compute-node registry; result intake; integration and human gates; operational telemetry; job-learning engine; control-plane API.
- **Must not:** own transfer, own canonical history, call models directly, or expose a remote shell.
- **Status:** scaffolded. Core migrates from the existing syncgate orchestrator in `ph3-kernel`; learning in `ph9-joblearn`.
- **Directives:** [aa-kernel.md](aa-kernel.md) · [aa-joblearn.md](aa-joblearn.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
