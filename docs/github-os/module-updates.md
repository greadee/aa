# Module Updates

A **module update** delivers a sub-problem on the phase branch that already owns the module, after that phase has merged. It reopens the retained branch instead of creating a new phase, so ownership, naming, and history stay with the module.

See [issues-and-subproblems.md](issues-and-subproblems.md) for how a problem is decomposed into sub-problems and how each is assigned to an owning branch.

## When To Use

Use a module update when:

- a sub-problem belongs to a module whose phase already merged;
- the work is additive or corrective within that module's existing boundary;
- a new phase would duplicate an existing one.

Do not create a new phase for work that belongs to an existing module and its branch. If the work needs a new boundary, it is a new phase, not a module update.

## Cycle

```text
reopen retained phase branch
  -> fast-forward it to main
  -> author the module update plan
  -> implement slices (one commit each)
  -> author the module update summary
  -> link the module update PR from the summary
  -> open a PR from the branch into main
  -> review
  -> merge, and retain the branch
```

## Module Update Plan

Authored at the **start** of the update, beside the module it changes:

```text
<module>/module-upd-plan.md
```

For a documentation-only update, place the plan with the documentation it changes (for example `docs/github-os/module-upd-plan.md`).

Content:

1. **Objective** — the outcome, stated so it can be verified.
2. **Starting state** — the ref the branch starts from, what exists, what is missing, and the constraints.
3. **Scope** — in scope and out of scope.
4. **Decisions** — decisions made for this update, each with reasoning.
5. **Slices** — each slice is independently validatable and maps to exactly one commit, with its exact commit message.
6. **Required end state** — verifiable outcomes.
7. **Exit criteria** — the definition of done.
8. **Test plan** — what will be tested and at which layer.

See the [module update plan template](templates/module/module-upd-plan.md).

## Module Update Summary

Authored at the **end** of the update, derived from the plan:

```text
<module>/module-upd-summary.md
```

Content: delivered slices and commits; what changed about the module and the app; whether this is part of a larger change and the umbrella it belongs to; validation with evidence; decisions affirmed; deviations with reasons; and follow-up.

Together the module update plan and summary are the durable record of the update, exactly as `plan.md` and `summary.md` are for a phase. See the [module update summary template](templates/module/module-upd-summary.md).

## Slices

A module update is sliced like a phase: each slice is independently validatable and maps to exactly one commit. Reserve the final slices for the summary and for linking the pull request, and name the exact commit message for every slice in the plan.

## Pull Request

Open one PR from the retained branch into `main`:

```text
<branch> -> main
```

Preferred title:

```text
Phase {N} module update: {topic} (aa-{module})
```

The body records the objective, the tracking links (sub-problem document, parent umbrella, module plan, module summary), progress, current status, known issues and risks, validation, and exit criteria.

## Branch Retention

The owning branch is retained after the update merges, exactly as after a phase. Never delete or rewrite it. See [git-workflow.md](git-workflow.md#branch-retention).

## Related

- Decomposition and sub-problem documents: [issues-and-subproblems.md](issues-and-subproblems.md)
- Phase lifecycle, milestones, and tracking issues: [phase-management.md](phase-management.md)
- Branch and commit conventions: [git-workflow.md](git-workflow.md)
- Pull requests and progress updates: [pull-requests.md](pull-requests.md)
- Documentation tree: [documentation-system.md](documentation-system.md)
