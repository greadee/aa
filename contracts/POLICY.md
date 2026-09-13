# Contract Versioning and Compatibility Policy

`aa-contracts` is the single source of truth for objects and protocols that cross a module boundary. This policy defines how those contracts evolve. It exists so that modules can be built, versioned, and replaced independently without silent breakage.

## Media and source of truth

- **JSON Schema (draft 2020-12)** under `schemas/v{MAJOR}/` is the source of truth for object shape.
- **RPC specification** under `rpc/` defines inter-module calls.
- **OpenAPI** under `openapi/` defines the external control-plane surface.
- Bindings under `go/`, `typescript/`, and `python/` are authored to match the schemas. They are not generated in phase 1; the conformance suite guards against drift.

## Versioning model

Contracts are versioned with `MAJOR.MINOR`:

- `MAJOR` is encoded in the path (`schemas/v1/`). A new major is a new directory; both directories may coexist during migration.
- `MINOR` is encoded in each document's `$id` and `x-contract-version` (for example `1.0`, `1.1`).

### Compatibility rules

| Change | Allowed in | Rule |
|---|---|---|
| Add an optional field | minor | Consumers must ignore unknown fields |
| Add a new enum member to a closed set | **major** | Treat closed enums as breaking |
| Add a new event type in a namespaced family | minor | Consumers must ignore unknown event types |
| Widen a numeric range | minor | Producers within the old range remain valid |
| Remove or rename a field | major | Requires a migration plan |
| Change a field's type or meaning | major | Requires a migration plan |
| Tighten validation (new required field, narrower range) | major | Old producers must keep working until migrated |
| Change a `$id` or title only | patch (no document change) | No compatibility impact |

Additive, backward-compatible changes bump `MINOR`. Breaking changes require a new `MAJOR` directory and a written migration note in this file.

### Unknown data

- Documents carry a `schema_version` (and where useful `contract_version`).
- Consumers **must** ignore unknown properties (`additionalProperties: true` at the open boundary) unless a schema explicitly closes an object.
- Producers **must not** rely on unknown fields surviving a round trip.

## Identifiers

- Identifiers are opaque strings matching `^[A-Za-z0-9][A-Za-z0-9_.:@+-]*$` unless a schema narrows this.
- Identifiers are stable and never reused. Superseding an entity creates a new version of it, not a new identity.
- Timestamps are RFC 3339 UTC. Ordering never depends on wall clock; ordering fields (`sequence`, `revision`) are explicit when order matters.

## Deprecation

1. Mark a field or enum member `deprecated: true` in the schema and document the replacement.
2. Keep it for at least one `MINOR` cycle.
3. Remove only in the next `MAJOR` with a migration note.

## Change process

1. Propose the change in a phase issue and record the decision in the phase `adr.md`.
2. Update the schema, then the bindings, then the conformance fixtures.
3. Update consumers in the same phase or document the migration if they cannot move immediately.
4. Run the conformance suite and the architecture boundary harness.

## Ownership

- `contracts` owns the schemas and their version numbers.
- A consuming module owns its own adapters to the contracts, never a private copy of the contract types.
- If two modules need different shapes for the same concept, the fix belongs in `contracts`, not in either module.

## Migration notes

_None yet. Breaking changes are recorded here with the major version that introduced them._
