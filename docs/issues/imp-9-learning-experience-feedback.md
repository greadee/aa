# ISS-IMP-9 — kernel/joblearn + memory: learning / experience feedback

**Type:** feature
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#31](https://github.com/greadee/aa/issues/31)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 5)
**Resolves:** A9 (apprenticing/studying learning states)

## Goal

Introduce/harden the learning loop without conflating accumulated experience with base-model
capability, separating **role** (expertise), **capability** (model/reasoning strength), and
**experience** (what the role/system has learned).

## Problem

`kernel/joblearn` attributes, scores, distills, and gates candidates, but there is no
experience-feedback loop (result → lesson → store → future retrieval/context) and no apprenticeship
model. A stronger model should teach a cheaper one indirectly through reusable rationale, review
feedback, corrections, examples, or lessons — without permanently pairing `senior` + `junior` agents.

## Requirements

- Experience feedback: execution → result/evaluation → reusable lesson → experience store → future
  retrieval/context.
- Selective mentorship/escalation pattern (not permanent duplicate roles).
- Learning states: **apprenticing** (project scope) and **studying** (organization scope), with
  provenance; lessons retain project provenance until deliberately promoted.
- Contamination/quality control: not every agent output becomes trusted learning; evidence gates and
  the existing promotion lifecycle apply.
- Deterministic, evidence-based; no model in the control path.

## Acceptance Criteria

- [x] Execution feedback produces reusable lessons stored as candidates with provenance.
- [x] Apprenticing/studying states represented with project/organization scope.
- [x] Selective mentorship/escalation modeled without permanent senior/junior role duplication.
- [x] Contamination controls and evidence gates enforced; promotion stays explicit via `memory`.
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`kernel/joblearn/experience` closes the loop: `Derive(policy, Observation)` turns
a **verified successful** result with review lessons into project-scope
`Candidate`s carrying provenance (work package, attempt, evidence).
`Policy.MinConfidence`/`RequireVerified`/`RequireEvidence` gate contamination —
anything weaker yields no lesson and a reason. Learning states are
`Apprenticing` (project) and `Studying` (organization); `Graduate` explicitly
promotes a project lesson to organization scope, retaining provenance. A
per-lesson `Mentorship` represents selective teaching (a stronger model shares
one lesson) without permanent role pairs. `Store` is the experience seam
(`MemStore` reference); promotion beyond CANDIDATE stays explicit in `memory`.
Deterministic and model-free.

Decision record: [ADR-0152](../adr/ADR-0152-experience-feedback-and-learning-states.md).

## Dependencies

Follows ISS-IMP-3/7/8. Related to ISS-IMP-10 (attribution/observability).
