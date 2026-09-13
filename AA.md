# AA.md

Primary agent directive for the **aa** repository.

This repository uses the **aa GitHub OS** (see [docs/github-os/](docs/github-os/README.md)). Do not load the entire documentation tree by default. Read only the module required for the current task.

Submodule directive files extend this one and live beside the module they govern, named `aa-<scope>.md` (for example `kernel/aa-kernel.md`, `console/aa-tui.md`, `obsv/aa-obsrv.md`). Read the module directive before changing that module. Aspect directives such as `aa-strgy.md` (strategy), `aa-issue.md` (issues), and `aa-joblearn.md` (job learning) live in the module that owns the aspect.

## Task Router

### If you are planning a phase or milestone
Read [docs/github-os/phase-management.md](docs/github-os/phase-management.md), [docs/github-os/phase-documentation.md](docs/github-os/phase-documentation.md), and [docs/github-os/sprint-planning.md](docs/github-os/sprint-planning.md).

### If you are writing a phase plan, UML, or summary
Read [docs/github-os/phase-documentation.md](docs/github-os/phase-documentation.md) and use [docs/github-os/templates/phase/](docs/github-os/templates/phase/).

### If you are changing Git history, branches, commits, or merge structure
Read [docs/github-os/git-workflow.md](docs/github-os/git-workflow.md) and, when rewriting history, [docs/github-os/repository-reconstruction.md](docs/github-os/repository-reconstruction.md).

### If you are implementing a tracked issue
Read the phase plan and issue docs, the relevant ADR, [docs/github-os/git-workflow.md](docs/github-os/git-workflow.md), and [docs/github-os/pull-requests.md](docs/github-os/pull-requests.md).

### If you are creating or updating GitHub issues
Read [docs/github-os/issues-and-stories.md](docs/github-os/issues-and-stories.md).

### If you are opening or updating pull requests
Read [docs/github-os/pull-requests.md](docs/github-os/pull-requests.md).

### If you are reviewing code
Read [docs/github-os/code-review.md](docs/github-os/code-review.md).

### If you are auditing the repository
Read [docs/github-os/repository-audits.md](docs/github-os/repository-audits.md).

### If you are updating project documentation
Read [docs/github-os/documentation-system.md](docs/github-os/documentation-system.md) and [docs/github-os/adrs-and-pitfalls.md](docs/github-os/adrs-and-pitfalls.md).

### If you are choosing a model for a task
Read [docs/github-os/model-selection.md](docs/github-os/model-selection.md).

### If you are changing architecture or module boundaries
Read [docs/architecture/README.md](docs/architecture/README.md) and the owning module's `aa-*.md`.

## Core Rules

1. Inspect before changing.
2. `main` is the stable integrated history.
3. Major work is grouped into sequential phase branches named identically to their phase folder (`ph{N}-{scope}`).
4. Every phase authors `plan.md` + `plan.uml` at the start and `summary.md` + `summary.uml` at the end.
5. A slice maps to exactly one commit; the plan names the exact commit message.
6. Module boundaries are enforced by contracts and architecture tests; modules never import each other's internals.
7. Planned work is visible in GitHub; findings that must survive the session become issues.
8. Significant issues and decisions get durable Markdown documentation.
9. Do not squash a major phase into one commit unless explicitly requested.
10. Use the least expensive model class that can reliably complete each slice.
11. GitHub should make project state understandable without private agent context.

## Commit Style

Default to concise, lowercase, action-first messages.

```text
add CI and docs-link-check workflow
adapt existing github os
add architecture documentation
scaffold module workspaces and agent directives
```

Conventional prefixes are allowed but optional. Do not mention the model merely because an agent made the change.
