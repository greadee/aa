# Git Workflow

## Git Philosophy

Treat Git history as part of project documentation.

History should show:

1. the major stages in which the project was built;
2. meaningful incremental work inside each stage;
3. fixes and refactors as they actually occurred;
4. the point at which each major stage was integrated into `main`.

Do not manufacture an artificially perfect history.

Do not squash substantial development into one commit merely for cosmetic cleanliness.

## Main Branch

`main` is the stable integration branch.

Major unfinished development should normally not happen directly on `main`.

`main` should primarily contain:

- completed phase merges;
- accepted milestone work;
- small corrections that genuinely do not warrant a development branch;
- repository-level maintenance where appropriate.

## Phase Branches

The phase branch name is **identical to the phase documentation folder name** under `docs/phases/`. This is a hard rule: if the folder is `ph0-scaffold`, the branch is `ph0-scaffold`.

Use:

```text
ph{number}-{scope}
```

Examples:

```text
ph0-scaffold
ph1-contracts
ph2-memory
ph5-forge
ph9-joblearn
ph10-ui
```

Rules:

- the branch name and the `docs/phases/ph{N}-{scope}` folder name must match exactly;
- start at `ph0` when a setup/foundation phase exists;
- do not zero-pad phase numbers;
- use a hyphen between the number and the scope;
- use a short, lowercase, kebab-case scope (`scaffold`, `contracts`, `memory`, `joblearn`, `visualizer`);
- phase branches represent milestones, not individual commits.

## Phase Lifecycle

```text
main
  ↓
create phase branch
  ↓
incremental development
  ↓
testing / fixes / refactoring
  ↓
documentation / finalization
  ↓
phase review / audit
  ↓
phase PR
  ↓
merge to main
```

## Issue Branches

Within a phase, use short-lived branches when they improve visibility and reviewability.

Preferred:

```text
<type>/<issue-number>-<short-description>
```

Examples:

```text
feature/123-broker-sync
fix/131-duplicate-transactions
refactor/136-ingestion-providers
test/142-broker-integration
docs/145-broker-architecture
```

Issue branches normally branch from the active phase branch and PR back into it.

Do not create a branch for every tiny correction.

Create one when work:

- corresponds to a separate tracked issue;
- should have its own review;
- changes a different subsystem;
- has independent acceptance criteria;
- is large enough to obscure another PR;
- can merge independently.

## Commit Philosophy

Commits should be understandable increments of actual development.

Prefer several meaningful commits over:

- one enormous phase-long commit;
- dozens of microscopic commits.

A commit may represent:

- introducing a component;
- implementing a service;
- adding an ingestion path;
- adding tests;
- fixing a discovered bug;
- changing a schema;
- refactoring a subsystem;
- adding documentation.

Closely related changes may be committed together.

Unrelated changes should normally be separated.

## Commit Message Style

Default:

```text
<action> <specific thing changed> [and <closely related change>]
```

Prefer concise, technical, action-first phrasing.

Examples:

```text
add basic cli and smoke test
fix missing print statement in cli loop test
add created_at, updated_at to position, refactor importer unit test
fix schema inconsistencies
remove unused path import in ingestion test file
add earnings calendar ingestion test and schema fixes
add live price streaming unit tests
improve test suite time and coverage
major refactor around ingestion pkg
finish broker-sync documentation and cli refactor
```

Common verbs:

```text
add
fix
remove
update
refactor
improve
finish
rename
move
implement
extend
set
clean
document
```

Conventional prefixes are allowed but optional:

```text
feat(db): add db connection and domain/storage models
fix(importer): change upsert order and fix bad assertions
test(importer): add unit tests
docs: add phase plan and starter ADRs
chore: update repository configuration
```

Do not force Conventional Commits across the repository.

## Message Tone

Messages should generally:

- fit on one line;
- name the affected subsystem or behavior;
- describe what changed;
- use codebase terminology freely;
- avoid verbose corporate phrasing.

Technical abbreviations are acceptable when familiar to the repository.

Avoid:

```text
updates
changes
misc
stuff
working
wip
codex changes
AI fixes
```

Do not mention the model merely because an agent made the change.

## Tests and Commits

Tests are first-class development work.

They may be:

1. in the same commit as the implementation they validate; or
2. in a separate follow-up commit when substantial.

Use whichever produces the clearest history.

## Refactors

Name the affected area.

Prefer:

```text
major refactor around ingestion pkg
refactor importer validation flow
```

Avoid:

```text
refactor code
```

## Documentation Commits

Documentation can ship:

- with the implementation it describes; or
- in a closely following docs commit when substantial.

## Merge Strategy

For major phase branches, prefer a normal merge that preserves the development commits.

Desired:

```text
phase branch:
A -- B -- C -- D -- E
                    \
main ---------------- M
```

Avoid replacing an entire phase with one giant squash commit unless explicitly requested.

## Branch Retention

Merged phase branches are **never deleted**. They are the durable, browsable work history of the phase and are kept on the remote after the phase PR merges.

Rules:

- Do not enable GitHub's "automatically delete head branches" for this repository.
- Do not delete a phase branch after merge, and do not rewrite it after merge.
- The merge into `main` is a normal merge commit that preserves the phase's individual commits (no squash, no rebase).
- If a phase needs correction after merge, add new commits on a new branch and open a new PR; never force-push the retained branch.
- The phase summary records the branch name, and the branch remains available for review and audit.

This is a deliberate exception to default Git-hosting cleanup behavior.
