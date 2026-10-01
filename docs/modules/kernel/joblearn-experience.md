# kernel/joblearn/experience

> Project history for the `kernel/joblearn/experience` submodule of [aa-kernel](./README.md).

**Responsibility** — Close the learning loop: execution → result/evaluation →
reusable lesson → experience store → future retrieval/context. Derivation is
deterministic and model-free; lessons are proposals with provenance.

**Source** — [`kernel/joblearn/experience/`](../../../kernel/joblearn/experience/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Feedback loop

`Derive(policy, Observation)` turns a **verified successful** execution result
with review lessons into project-scope `joblearn.Candidate`s (level `project`,
scope `project:<id>`), each carrying `Provenance` with the work package, attempt,
and observation evidence. Contamination is controlled: an unverified outcome, a
non-success, confidence below `Policy.MinConfidence`, or missing evidence yields
no lesson (with a reason).

- **Learning states** — `Apprenticing` (project scope) / `Studying`
  (organization scope) / `Practicing`; `Graduate` explicitly upgrades a project
  lesson to organization scope, retaining provenance.
- **Selective mentorship** — `Derive` may attach a per-lesson `Mentorship`
  (a stronger model shares one lesson with cheaper workers). It is never a
  permanent senior/junior role pairing.
- **Experience store** — `Store` is the seam to memory; `MemStore` is the
  in-memory reference. Promotion beyond CANDIDATE stays explicit in `memory`.

Decision record: [ADR-0152](../../adr/ADR-0152-experience-feedback-and-learning-states.md).
