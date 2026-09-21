# GitHub OS

The aa process layer. GitHub is the durable **execution** record; this `docs/` tree is the durable **knowledge** record; agent conversations are temporary context.

Read only the module required for the current task. The task router lives in [../../AA.md](../../AA.md).

## Modules

| Module | Use when |
|---|---|
| [phase-management.md](phase-management.md) | Planning a phase, milestone, tracking issue, or phase PR |
| [phase-documentation.md](phase-documentation.md) | Creating phase plans, UML, summaries; branch/folder naming |
| [sprint-planning.md](sprint-planning.md) | Breaking a phase into slices and commits |
| [issues-and-stories.md](issues-and-stories.md) | Creating or updating issues, stories, and bugs |
| [issues-and-subproblems.md](issues-and-subproblems.md) | Decomposing a cross-branch problem into sub-problems and owning branches |
| [module-updates.md](module-updates.md) | Reopening an owning branch to deliver a sub-problem after its phase merged |
| [pull-requests.md](pull-requests.md) | Opening or updating PRs, including the Draft phase PR |
| [code-review.md](code-review.md) | Reviewing code and producing findings |
| [repository-audits.md](repository-audits.md) | Auditing the repository or a phase end |
| [repository-reconstruction.md](repository-reconstruction.md) | Rebuilding or rewriting Git history |
| [adrs-and-pitfalls.md](adrs-and-pitfalls.md) | Recording decisions and expensive-to-rediscover pitfalls |
| [documentation-system.md](documentation-system.md) | Updating the documentation tree and indexes |
| [git-workflow.md](git-workflow.md) | Branching, committing, and merging |
| [model-selection.md](model-selection.md) | Choosing a model class for a slice |

## Templates

- `templates/phase/` — phase `plan.md`, `plan.uml`, `summary.md`, `summary.uml`, `adr.md`, and issue docs
- `templates/github/` — tracking issue, PRs, issues, audit findings, progress updates
- `templates/feature/` — feature README, implementation, testing, and pitfalls
- `templates/issues/` — umbrella problem and sub-problem documents
- `templates/module/` — module update `plan` and `summary`

## Core rules

1. Inspect before changing.
2. `main` is the stable integrated history.
3. Major work is grouped into sequential phase branches.
4. Branch names equal phase folder names (`ph{N}-{scope}`).
5. Every phase authors a plan and a summary, each with its own UML.
6. A slice maps to exactly one commit.
7. A planned feature is not complete until its tests, docs, and audit are.
8. Record important architectural decisions and their reasoning.
9. Record expensive-to-rediscover pitfalls and solutions.
10. Use the least expensive model class that can reliably complete each slice.
11. Findings that must survive the session become issues.
12. GitHub should make project state understandable without private agent context.
13. A problem that spans branches is decomposed into sub-problems, each owned by exactly one branch.
14. Work that belongs to a merged phase is delivered as a module update on its retained branch, never as a new phase.
