# Contributing to aa

Thanks for helping. This repository is a modular Go/Python/TypeScript monorepo
with a documentation and GitHub process that is part of the implementation.

Read the [documentation entry point](docs/index/README.md) first, then the module
directive beside the code you are changing (`aa-*.md`).

## Setup

Requirements:

- Go 1.26
- Python 3.12+
- Node 20+ (TypeScript contracts only)
- Git and GitHub CLI (`gh`)

The repository uses a Go workspace, so no `go mod download` step is required at
the root.

```sh
git clone https://github.com/greadee/aa.git
cd aa
go version
go run ./app selfcheck
```

## Quickstart

```sh
# deterministic composition check; no model or network required
go run ./app selfcheck

# control plane on loopback
go run ./app serve

# console against the running control plane (separate shell)
go run ./app console

# offline console using the deterministic fake client
go run ./app console --fake

# read-only web surface
go run ./app console --web
```

`aa serve` requires a bearer token on every path except `/health`; set
`AA_CONTROL_TOKEN` or pass `--token`. The empty token is only for local
development.

## Branch and commit rules

- `main` is the stable integrated history. Implementation lands on `dev` first.
- One coherent slice or issue maps to one commit with an action-first, lowercase
  message, for example `add docs foundation templates`.
- Keep commits readable. Do not squash a phase or sprint into one commit.
- Never commit secrets, local state (`.aa/`), build output, or generated caches.
- Create a branch per issue or sprint; open the PR against the correct base
  (`dev` for sprint work, the phase branch for a module update).

## Validation

Run the checks that cover the code you changed. At minimum:

```sh
gofmt -l .                       # must print nothing
cd tools && go run ./cmd/archtest -root .. && go test ./...
```

For a Go module:

```sh
cd <module>
go build ./... && go vet ./... && go test ./...
```

For contracts:

```sh
python -m compileall contracts/python/aa_contracts
python contracts/tools/check_binding_parity.py
python -m unittest discover -s contracts/python/tests -t contracts/python
python -m unittest discover -s contracts/tools -p "test_*.py"
```

For inference:

```sh
cd runtime/inference
ruff format --check . && ruff check . && mypy && pytest
```

For documentation:

```sh
lychee --offline --include-fragments "./**/*.md"
```

Record the commands and results on the PR. A claim without evidence is not
validation.

## Definitions of done

Every issue and PR follows the [Definitions of Done](docs/github-os/definitions-of-done.md).
In short: observable acceptance criteria, tests or reproducible validation,
updated docs where behavior changes, linked issues, green required checks, and a
closeout comment with evidence.

## Issues

Use the GitHub issue templates. Include:

- goal and value;
- requirements stated as behavior;
- observable acceptance criteria;
- dependencies and blockers;
- severity and area labels;
- a linked follow-up issue for anything deferred.

Severity follows the repository audit policy: P0/P1 block a sprint; P2 is fixed
or explicitly deferred; P3 is deferred with a linked issue.

## Pull requests

- Link every issue the PR closes or advances.
- State the purpose, changes, validation, failure paths, and documentation impact.
- Keep unrelated changes out of the PR.
- Use the PR dashboard when it carries multiple issues, and update it as work
  completes.
- Comment on blockers, scope changes, and material decisions immediately, in
  chronological order.
- Mark the PR ready only when known work is described and required checks pass.

Templates live in `.github/ISSUE_TEMPLATE/`, `.github/PULL_REQUEST_TEMPLATE.md`,
and [docs/github-os/templates/](docs/github-os/templates/).

## Documentation

Documentation is part of implementation. Update the module docs, indexes, and
current-state page when behavior, contracts, workflow, or state changes. Preserve
historical phase/update records; add a dated supersession note instead of
rewriting history.

## License

By contributing you agree that your contribution is licensed under the repository
[MIT license](LICENSE.md).
