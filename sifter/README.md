# aa-sifter

Decode a prompt and route it to the appropriate local or cloud model under budget and approval.

- **Owns:** classification, human approval gate, preflight, policy routing, budgets, escalation, provider abstraction (Ollama/OpenAI-compatible), verification, context compression/handoff, compute orchestration across local/cloud.
- **Must not:** hold project state or perform orchestration.
- **Status:** scaffolded. Implementation migrates from the existing `compute-sifter` Python source in `ph4-sifter`.
- **Directive:** [aa-sifter.md](aa-sifter.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
