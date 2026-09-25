# Terminology, state registry, and event taxonomy

Canonical vocabulary for **aa**. If a term is ambiguous across modules, this document is the tie-breaker.

## 1. Organizational terms

| Term | Meaning |
|---|---|
| **Role** | A durable organizational responsibility with defined authority and artifacts (e.g. Architect, Inspector). Roles outlive models and projects. |
| **Trade** | A durable capability that can be instantiated to perform work (e.g. Backend, Test/QA, Reviewer). A trade is provider-independent. |
| **Worker** | An instantiation of a trade with a concrete provider, runtime, tool policy, and instruction version. Workers are ephemeral per assignment. |
| **Model** | An execution backend for a worker. Never an identity. |
| **Work package** | A bounded, verifiable unit of engineering work assigned to a worker. |
| **Assignment** | The binding of a work package to a worker under a lease and execution contract. |
| **Attempt** | One execution of an assignment, producing a result envelope. |
| **Artifact** | A produced, typed, hash-verifiable output (patch, report, candidate, etc.). |
| **Deficiency** | A verified shortfall against acceptance criteria that requires repair. |
| **Checkpoint** | A durable marker in project history (phase, milestone, or projection state). Context determines which meaning; the specific kind is always stated. |

## 2. Platform terms

| Term | Meaning |
|---|---|
| **Authority** | The single node permitted to create canonical records and dispatch execution for a project. |
| **Replica** | A node that observes and syncs but cannot write canonical records or execute with authority. |
| **System of record** | The canonical, portable set of project records. Owned by `aa-memory`. |
| **Projection** | A rebuildable, non-authoritative view derived from canonical records (e.g. SQLite). |
| **Memory** | Validated knowledge promoted from history; hierarchical and lifecycle-managed. |
| **Context bundle** | A deterministic, bounded compilation of memory and inputs handed to a worker. |
| **Execution contract** | The immutable, least-privilege authority under which an attempt runs. |
| **Result envelope** | The untrusted, structured output of an attempt, accepted only after validation. |
| **Gate** | A deterministic check or human decision that work must pass before advancing. |
| **Learning candidate** | A proposed piece of knowledge awaiting validation before promotion into memory. |

## 3. Memory hierarchy

```
Session  →  Task / Workstream  →  Project  →  Role / Trade  →  Workforce
```

Lifecycle for every candidate:

```
EPHEMERAL → CANDIDATE → VALIDATED → ACTIVE → SUPERSEDED → ARCHIVED
```

Promotion is deterministic: candidates require evidence, validation, and an explicit promotion rule. Only `aa-memory` stores canonical, permanent knowledge.

## 4. State registry

Each state family has exactly one owner. No state name is reused across families with a different meaning.

### Project states — owner: aa-memory
`CREATED`, `DISCOVERY`, `PLANNING`, `EXECUTING`, `VERIFYING`, `COMPLETE`, `BLOCKED`, `FAILED`, `CANCELLED`, `PAUSED`

### Work package states — owner: aa-memory
`PROPOSED`, `READY`, `QUEUED`, `RUNNING`, `COMPLETE`, `VERIFYING`, `VERIFIED`, `DEFICIENT`

### Assignment states — owner: aa-kernel
`planned`, `leased`, `preparing`, `running`, `paused`, `collecting`, `awaiting_gates`, `accepted`, `failed`, `canceled`, `expired`

### Readiness states — owner: aa-kernel
`blocked`, `ready`, `dispatched`, `satisfied`

### Issue states — owner: aa-forge
`open`, `in_progress`, `review`, `closed`, `deferred`, `cancelled`

### Sprint/progress states — owner: process
`Planned`, `Ready`, `In Progress`, `Blocked`, `Review`, `Complete`, `Deferred`

## 5. Event taxonomy

Cross-module events use one vocabulary. There is **no** `AGENT_*` state family.

### Project and planning
`PROJECT_REGISTERED`, `PROJECT_UPDATED`, `WORK_PACKAGE_CREATED`, `WORK_PACKAGE_READY`, `WORK_PACKAGE_COMPLETED`, `DEPENDENCY_GRAPH_REVISED`

### Execution
`EXECUTION_PLANNED`, `EXECUTION_LEASED`, `EXECUTION_STARTED`, `EXECUTION_PAUSED`, `EXECUTION_RESUMED`, `EXECUTION_COMPLETED`, `EXECUTION_FAILED`, `EXECUTION_EXPIRED`

### Results and gates
`RESULT_RECORDED`, `TEST_RECORDED`, `REVIEW_RECORDED`, `HANDOFF_CREATED`, `ARTIFACT_RECORDED`, `WORK_ACCEPTED`, `WORK_REJECTED`, `DEFICIENCY_RAISED`

### Memory and learning
`TELEMETRY_RECORDED`, `MEMORY_CANDIDATE_CREATED`, `MEMORY_VALIDATED`, `MEMORY_PROMOTED`, `MEMORY_SUPERSEDED`, `MEMORY_ARCHIVED`

### Observation (aa-obsv protocol v1)
The observation vocabulary is defined by `aa-obsv` and mirrored in `aa-contracts`. It covers session, tool, file, and work-delta events with an explicit `source_type` and `source_confidence` (exact, correlated, observed, inferred). Confidence is semantic and must never be promoted.

## 6. Module scope names

Used for branches, folders, and directive files.

| Module | Scope token |
|---|---|
| aa-contracts | `contracts` |
| aa-registry | `registry` |
| aa-obsv | `obsv` |
| aa-kernel | `kernel` |
| aa-memory | `memory` |
| aa-sync | `sync` |
| aa-sifter | `sifter` |
| aa-forge | `forge` |
| aa-toolbox | `toolbox` |
| aa-visualizer | `visualizer` |
| aa-console | `console` |
