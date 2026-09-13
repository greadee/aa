# aa-contracts.md

Agent directive for the `contracts` module (aa-contracts).

## Responsibility

Define every object and protocol that crosses a module boundary, exactly once, and generate bindings from it.

## Owns

- JSON Schemas for all cross-module objects.
- The RPC/transport specification.
- Generated Go, TypeScript, and Python bindings.
- The conformance suite.
- Schema versioning and compatibility policy.
- The control-plane OpenAPI document.

## Must Not

- Contain business logic.
- Depend on any other module.
- Allow modules to hand-roll cross-module DTOs.

## Interfaces

- Generated packages consumed by every module.
- Schemas consumed by tooling and tests.

## Rules

1. Contracts depend on nothing; everything may depend on contracts.
2. Additive, backward-compatible changes bump the minor version; breaking changes require a major version and a migration plan.
3. Every generated binding is validated by the conformance suite.
4. If two modules model the same concept differently, the fix is in contracts, not in the modules.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Terminology and event taxonomy](../docs/reference/terminology.md)
