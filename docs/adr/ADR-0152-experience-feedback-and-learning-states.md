# ADR-0152 — Experience feedback and learning states

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 9](../issues/imp-9-learning-experience-feedback.md) (A9) |
| **Scope** | update phase · issue-impl-sep28 (Stage 5) |

## Context

`kernel/joblearn` attributed, scored, distilled, and gated candidates, but there
was no experience-feedback loop (result → lesson → store → future
retrieval/context) and no apprenticeship model. The design must separate **role**
(expertise), **capability** (model strength), and **experience** (what has been
learned), and must not conflate accumulated experience with base-model capability.

## Decision

- **Closed feedback loop.** `kernel/joblearn/experience.Derive` turns a verified
  successful result plus review lessons into project-scope `Candidate`s with
  provenance (work package, attempt, evidence). `Store` is the seam to memory for
  future retrieval/context.
- **Learning states.** `Apprenticing` (project scope) and `Studying`
  (organization scope), with `Practicing` for graduated roles. `Graduate`
  explicitly promotes a project lesson to organization scope, retaining
  provenance; lessons stay project-scoped until then.
- **Selective mentorship.** A stronger model may teach a cheaper one indirectly
  through one reusable lesson (`Mentorship`), modeled per lesson — never a
  permanent senior/junior role pairing and never a role duplication.
- **Contamination control.** Only a verified success with confidence ≥ the
  policy threshold and evidence yields lessons; otherwise nothing is learned
  (with a reason). Candidates remain proposals; promotion stays explicit in
  `memory` under the existing lifecycle and evidence gates.
- **Deterministic and model-free.** Derivation calls no model.

## Rationale

- Keeping experience separate from role and capability prevents a model upgrade
  from being mistaken for learned knowledge, and vice versa.
- Scoped learning states make provenance explicit and let promotion be deliberate.
- Gating on verified evidence resists knowledge poisoning.

## Alternatives and consequences

- **Permanent senior/junior role pairs:** rejected — duplicates roles and
  entrenches mentorship instead of transferring learning.
- **Learn from every output:** rejected — contamination risk; evidence gates are
  mandatory.
- **Auto-promote lessons:** rejected — promotion is memory's explicit lifecycle.
- Consequence: the experience store is a seam; wiring it into retrieval/context
  uses the existing candidate→memory promotion path.

## Related

- [experience module doc](../modules/kernel/joblearn-experience.md)
- [ADR-0115](./ADR-0115-every-capability-sits-behind-an-explicit-evidence-gate-and-is.md)
- [ADR index](./README.md)
