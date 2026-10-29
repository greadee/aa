# Sprint 2 — durable multi-project state and control-plane correctness

## Delivered

- One durable model for projects, graphs, work packages, assignments, and
  approvals (`app/internal/projectstate`).
- Project-scoped work submission/listing, governed not-found and conflict
  responses, and multi-project isolation tests.
- Serialized approval decisions with an `awaiting_gates` requirement; approve
  and reject are distinct and durable across restart.
- Fail-closed validation for corrupt project, work-package, assignment, and
  approval records.
- Scheduler assignment persistence errors are observable, and restart restores
  attempts/leases without overwriting history.
- OpenAPI documents the real approval/project/work/graph responses, with a
  spec-driven route and response-shape gate.
- Console fake matches server semantics and supports reject.
- Toolbox host replays deep-copied results; metrics have a structured-log
  exporter and documented compatibility breaks.

## Follow-ups

- Field-level cross-language binding parity remains a Phase 12 hardening item
  (ISS-CONTRACTS-1, #76).
- Scheduler execution/E2E and CLI/TUI completion remain Sprint 3 work.

## Validation

```sh
gofmt -l .
for m in contracts registry runtime obsv kernel memory sync forge toolbox visualizer ui app; do
  (cd "$m" && go build ./... && go vet ./... && go test ./...)
done
(cd tools && go test ./... && go run ./cmd/archtest -root ..)
(cd app && go run . selfcheck)
```
