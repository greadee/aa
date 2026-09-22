"""Python bindings for aa contract schemas v1. Standard library only.

The JSON Schema files under contracts/schemas/v1 are the source of truth.
Objects are represented as plain dictionaries and validated by :func:`decode`.
``TypedDict`` definitions provide static shape for editors and type checkers.
"""

from __future__ import annotations

import json
import re
from typing import Any, Callable, Dict, List, Optional, TypedDict

CONTRACT_VERSION = "1.0"

_IDENTIFIER_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.:@+-]*$")
_VERSION_RE = re.compile(r"^[0-9]+\.[0-9]+$")
_HASH_RE = re.compile(r"^[a-f0-9]{64}$")

PROJECT_STATES = {"CREATED", "DISCOVERY", "PLANNING", "EXECUTING", "VERIFYING",
                  "COMPLETE", "BLOCKED", "FAILED", "CANCELLED", "PAUSED"}
WORK_PACKAGE_STATES = {"PROPOSED", "READY", "QUEUED", "RUNNING",
                       "COMPLETE", "VERIFYING", "VERIFIED", "DEFICIENT"}
ASSIGNMENT_STATES = {"planned", "leased", "preparing", "running", "paused", "collecting",
                     "awaiting_gates", "accepted", "failed", "canceled", "expired"}
READINESS_STATES = {"blocked", "ready", "dispatched", "satisfied"}
ISSUE_STATES = {"open", "in_progress", "review", "closed", "deferred", "cancelled"}
MEMORY_LIFECYCLES = {"EPHEMERAL", "CANDIDATE", "VALIDATED", "ACTIVE", "SUPERSEDED", "ARCHIVED"}
EVENT_TYPES = {
    "PROJECT_REGISTERED", "PROJECT_UPDATED", "WORK_PACKAGE_CREATED", "WORK_PACKAGE_READY",
    "WORK_PACKAGE_COMPLETED", "DEPENDENCY_GRAPH_REVISED",
    "EXECUTION_PLANNED", "EXECUTION_LEASED", "EXECUTION_STARTED", "EXECUTION_PAUSED",
    "EXECUTION_RESUMED", "EXECUTION_COMPLETED", "EXECUTION_FAILED", "EXECUTION_EXPIRED",
    "RESULT_RECORDED", "TEST_RECORDED", "REVIEW_RECORDED", "HANDOFF_CREATED",
    "ARTIFACT_RECORDED", "WORK_ACCEPTED", "WORK_REJECTED", "DEFICIENCY_RAISED",
    "TELEMETRY_RECORDED", "MEMORY_CANDIDATE_CREATED", "MEMORY_VALIDATED",
    "MEMORY_PROMOTED", "MEMORY_SUPERSEDED", "MEMORY_ARCHIVED",
}
CAPABILITIES = {
    "read_project", "write_workspace", "execute_command", "run_tests", "network_access",
    "install_dependencies", "call_model", "read_secrets", "create_artifact",
    "open_pull_request", "merge", "deploy",
}
ROUTING_POLICIES = {"local_only", "cloud_only", "local_first", "expert_first", "adaptive", "budget_constrained"}
ACTOR_KINDS = {"human", "agent", "system", "service"}
TRACE_PHASES = {"observe", "plan", "model", "tool", "edit", "test", "gate", "verify", "integrate"}
TRACE_STEP_OUTCOMES = {"succeeded", "failed", "skipped", "blocked"}


def _identifier(field: str, value: Any) -> None:
    if not value:
        raise ValueError(f"{field}: identifier is required")
    if len(value) > 256 or not _IDENTIFIER_RE.match(value):
        raise ValueError(f"{field}: invalid identifier {value!r}")


def _version(field: str, value: Any) -> None:
    if not isinstance(value, str) or not _VERSION_RE.match(value):
        raise ValueError(f"{field}: invalid contract version {value!r}")


def _enum(field: str, value: Any, allowed) -> None:
    if value not in allowed:
        raise ValueError(f"{field}: invalid value {value!r}")


def _required(field: str, value: Any) -> None:
    if value is None or value == "":
        raise ValueError(f"{field}: is required")


def _optional_hash(field: str, value: Any) -> None:
    if value is not None and not _HASH_RE.match(value):
        raise ValueError(f"{field}: invalid sha256 hash")


def _check_references(data: Dict[str, Any], key: str) -> None:
    for i, ref in enumerate(data.get(key) or []):
        _required(f"{key}[{i}].kind", ref.get("kind"))
        _identifier(f"{key}[{i}].id", ref.get("id"))


def _validate_event(data: Dict[str, Any]) -> None:
    _identifier("aggregate.id", data["aggregate"].get("id"))
    _enum("aggregate.kind", data["aggregate"].get("kind"),
          {"project", "work_package", "assignment", "attempt", "artifact",
           "issue", "strategy", "memory", "tool", "workflow"})
    _enum("actor.kind", data["actor"].get("kind"), {"human", "agent", "system", "service"})
    _identifier("actor.id", data["actor"].get("id"))


def _validate_work_package(data: Dict[str, Any]) -> None:
    if not data.get("acceptance"):
        raise ValueError("acceptance: at least one criterion is required")


def _validate_task_graph(data: Dict[str, Any]) -> None:
    if data.get("revision", -1) < 0:
        raise ValueError("revision: is required and must be >= 0")
    for i, node in enumerate(data.get("nodes") or []):
        _identifier(f"nodes[{i}].workPackageId", node.get("workPackageId"))
        _enum(f"nodes[{i}].readiness", node.get("readiness"), READINESS_STATES)


def _validate_execution_contract(data: Dict[str, Any]) -> None:
    for i, cap in enumerate(data.get("capabilities") or []):
        _enum(f"capabilities[{i}]", cap, CAPABILITIES)
    for i, cap in enumerate(data.get("denied") or []):
        _enum(f"denied[{i}]", cap, CAPABILITIES)


def _validate_result(data: Dict[str, Any]) -> None:
    for i, art in enumerate(data.get("artifacts") or []):
        _optional_hash(f"artifacts[{i}].hash", art.get("hash"))
    for i, test in enumerate(data.get("tests") or []):
        _enum(f"tests[{i}].outcome", test.get("outcome"), {"passed", "failed", "skipped", "error"})


def _validate_memory_record(data: Dict[str, Any]) -> None:
    _enum("level", data.get("level"), {"session", "task", "project", "role", "workforce"})
    content = data.get("content") or {}
    _required("content.summary", content.get("summary"))
    if data.get("confidence") is not None and not 0 <= data["confidence"] <= 1:
        raise ValueError("confidence: must be between 0 and 1")


def _validate_issue(data: Dict[str, Any]) -> None:
    _enum("type", data.get("type"),
          {"feature", "user_story", "bug", "tech_debt", "refactor", "test",
           "documentation", "audit", "investigation", "deficiency"})


def _validate_route_request(data: Dict[str, Any]) -> None:
    _required("prompt", data.get("prompt"))
    if data.get("policy") is not None:
        _enum("policy", data["policy"], ROUTING_POLICIES)


def _validate_route_response(data: Dict[str, Any]) -> None:
    _required("requestId", data.get("requestId"))
    _enum("route", data.get("route"), {"local", "hybrid", "cloud", "blocked"})
    _enum("decisionLevel", data.get("decisionLevel"), {"routine", "significant", "major", "critical"})


def _validate_tool_manifest(data: Dict[str, Any]) -> None:
    _enum("toolKind", data.get("toolKind"), {"tool", "plugin", "mcp", "builtin"})
    if not data.get("capabilities"):
        raise ValueError("capabilities: at least one is required")


def _validate_trace(data: Dict[str, Any]) -> None:
    steps = data.get("steps") or []
    if not steps:
        raise ValueError("steps: at least one step is required")
    for i, step in enumerate(steps):
        seq = step.get("sequence")
        if not isinstance(seq, int) or isinstance(seq, bool) or seq < 0:
            raise ValueError(f"steps[{i}].sequence: is required and must be >= 0")
        _enum(f"steps[{i}].phase", step.get("phase"), TRACE_PHASES)
        _enum(f"steps[{i}].outcome", step.get("outcome"), TRACE_STEP_OUTCOMES)
        _optional_hash(f"steps[{i}].inputHash", step.get("inputHash"))
        _optional_hash(f"steps[{i}].outputHash", step.get("outputHash"))


def _validate_workflow(data: Dict[str, Any]) -> None:
    _required("version", data.get("version"))
    steps = data.get("steps") or []
    if not steps:
        raise ValueError("steps: at least one step is required")
    seen = set()
    for i, step in enumerate(steps):
        _identifier(f"steps[{i}].id", step.get("id"))
        if step.get("id") in seen:
            raise ValueError(f"steps[{i}].id: duplicate id {step.get('id')!r}")
        seen.add(step.get("id"))
        _enum(f"steps[{i}].kind", step.get("kind"),
              {"task", "tool", "gate", "human_approval", "parallel", "conditional"})


# Required fields per kind (all kinds also carry contractVersion and id).
_REQUIRED: Dict[str, List[str]] = {
    "event": ["sequence", "occurredAt", "type", "aggregate", "actor"],
    "work_package": ["title", "state", "acceptance"],
    "task_graph": ["projectId", "nodes", "revision"],
    "execution_contract": ["workPackageId", "assignmentId", "capabilities", "budget"],
    "result_envelope": ["attemptId", "assignmentId", "workPackageId", "status"],
    "telemetry": ["attemptId", "metricVersion", "outcome"],
    "trace": ["attemptId", "steps"],
    "project_record": ["name", "state", "layoutVersion"],
    "memory_record": ["level", "lifecycle", "content"],
    "issue": ["title", "type", "status"],
    "strategy": ["title", "lifecycle", "guidance"],
    "route_request": ["prompt"],
    "route_response": ["requestId", "route", "decisionLevel", "requiresHumanApproval"],
    "tool_manifest": ["name", "toolKind", "version", "capabilities"],
    "workflow": ["version", "steps"],
}

_ENUMS: Dict[str, Dict[str, set]] = {
    "event": {"type": EVENT_TYPES},
    "work_package": {"state": WORK_PACKAGE_STATES, "risk": {"low", "moderate", "high", "critical"}},
    "execution_contract": {},
    "result_envelope": {"status": {"succeeded", "failed", "partial", "blocked", "cancelled"}},
    "telemetry": {"outcome": {"succeeded", "failed", "partial", "blocked", "cancelled", "unknown"}},
    "trace": {},
    "project_record": {"state": PROJECT_STATES},
    "memory_record": {"lifecycle": MEMORY_LIFECYCLES},
    "issue": {"status": ISSUE_STATES, "severity": {"P0", "P1", "P2", "P3", "none"}},
    "strategy": {"lifecycle": MEMORY_LIFECYCLES},
    "route_response": {"route": {"local", "hybrid", "cloud", "blocked"},
                        "decisionLevel": {"routine", "significant", "major", "critical"}},
}

_EXTRA: Dict[str, Callable[[Dict[str, Any]], None]] = {
    "event": _validate_event,
    "work_package": _validate_work_package,
    "task_graph": _validate_task_graph,
    "execution_contract": _validate_execution_contract,
    "result_envelope": _validate_result,
    "memory_record": _validate_memory_record,
    "issue": _validate_issue,
    "route_request": _validate_route_request,
    "route_response": _validate_route_response,
    "tool_manifest": _validate_tool_manifest,
    "workflow": _validate_workflow,
    "trace": _validate_trace,
}


def validate(data: Dict[str, Any]) -> Dict[str, Any]:
    """Validate a contract object and return it. Raises ValueError on failure."""
    if not isinstance(data, dict):
        raise ValueError("contract object must be a mapping")
    kind = data.get("kind")
    if kind not in _REQUIRED:
        raise ValueError(f"kind: unknown contract kind {kind!r}")
    _version("contractVersion", data.get("contractVersion"))
    _identifier("id", data.get("id"))
    for field in _REQUIRED[kind]:
        _required(field, data.get(field))
    for field, allowed in _ENUMS.get(kind, {}).items():
        if data.get(field) is not None:
            _enum(field, data[field], allowed)
    _EXTRA.get(kind, lambda _d: None)(data)
    return data


def decode(data: Dict[str, Any]) -> Dict[str, Any]:
    """Validate and return a contract object."""
    return validate(data)


def encode(data: Dict[str, Any]) -> Dict[str, Any]:
    """Validate and return a contract object (JSON-serializable)."""
    return validate(data)


def loads(text: str) -> Dict[str, Any]:
    """Parse and validate a JSON contract object."""
    return decode(json.loads(text))


def dumps(data: Dict[str, Any], indent: Optional[int] = 2) -> str:
    """Validate and serialize a contract object to JSON."""
    return json.dumps(encode(data), indent=indent, sort_keys=False)


# ---------------------------------------------------------------------------
# Static shape (TypedDict). Runtime validation is performed by validate().
# ---------------------------------------------------------------------------

class Reference(TypedDict, total=False):
    kind: str
    id: str
    version: str


class Actor(TypedDict, total=False):
    kind: str
    id: str
    role: str
    trade: str
    workerId: str
    model: str


class Budget(TypedDict, total=False):
    maxCalls: int
    maxTokens: int
    maxCostUsd: float
    maxDurationSeconds: int


class Provenance(TypedDict, total=False):
    source: str
    producedAt: str
    producedBy: Actor
    confidence: str
    evidence: List[Reference]
    contentHash: str


class Envelope(TypedDict, total=False):
    contractVersion: str
    kind: str
    id: str
    projectId: str
    createdAt: str
    revision: int
    provenance: Provenance


class EventAggregate(TypedDict, total=False):
    kind: str
    id: str
    version: int


class Event(Envelope, total=False):
    sequence: int
    occurredAt: str
    recordedAt: str
    type: str
    aggregate: EventAggregate
    actor: Actor
    causationId: str
    correlationId: str
    payload: Dict[str, Any]


class WorkPackage(Envelope, total=False):
    title: str
    description: str
    state: str
    trade: str
    role: str
    dependencies: List[str]
    inputs: List[Reference]
    deliverables: List[str]
    acceptance: List[str]
    budget: Budget
    risk: str


class TaskGraph(Envelope, total=False):
    projectId: str
    revision: int
    nodes: List[Dict[str, Any]]
    digest: str


class ExecutionContract(Envelope, total=False):
    workPackageId: str
    assignmentId: str
    capabilities: List[str]
    denied: List[str]
    runtime: Dict[str, Any]
    workspace: Dict[str, Any]
    budget: Budget
    gates: List[str]
    expiresAt: str
    digest: str


class ResultEnvelope(Envelope, total=False):
    attemptId: str
    assignmentId: str
    workPackageId: str
    status: str
    summary: str
    artifacts: List[Dict[str, Any]]
    tests: List[Dict[str, Any]]
    telemetryRef: Reference
    failure: Dict[str, Any]
    producedBy: Actor


class Telemetry(Envelope, total=False):
    attemptId: str
    assignmentId: str
    metricVersion: str
    outcome: str
    calls: Dict[str, int]
    tokens: Dict[str, int]
    costUsd: float
    durationMs: int
    resources: Dict[str, Any]
    versions: Dict[str, str]


class TraceStep(TypedDict, total=False):
    sequence: int
    at: str
    phase: str
    actor: Actor
    operation: str
    target: Reference
    inputHash: str
    outputHash: str
    outcome: str
    errorClass: str
    durationMs: int
    tokens: Dict[str, int]
    costUsd: float
    redacted: bool
    evidence: List[Reference]


class Trace(Envelope, total=False):
    attemptId: str
    assignmentId: str
    workPackageId: str
    redactionVersion: str
    truncated: bool
    steps: List[TraceStep]


class ProjectRecord(Envelope, total=False):
    name: str
    state: str
    layoutVersion: str
    authorityId: str
    replicaIds: List[str]
    records: List[Reference]
    digest: str


class MemoryRecord(Envelope, total=False):
    level: str
    lifecycle: str
    title: str
    content: Dict[str, Any]
    applicability: Dict[str, Any]
    evidence: List[Reference]
    confidence: float
    supersedes: str


class Issue(Envelope, total=False):
    title: str
    type: str
    status: str
    severity: str
    body: str
    milestone: str
    forgeRef: Reference
    links: Dict[str, Any]
    deferredTo: str


class Strategy(Envelope, total=False):
    title: str
    lifecycle: str
    guidance: str
    whenToUse: str
    whenNotToUse: str
    applicability: Dict[str, Any]
    evidence: List[Reference]
    supersedes: str


class RouteRequest(Envelope, total=False):
    attemptId: str
    prompt: str
    context: str
    policy: str
    budget: Budget
    metadata: Dict[str, Any]


class RouteResponse(Envelope, total=False):
    requestId: str
    route: str
    decisionLevel: str
    plannerTier: str
    executorTier: str
    reviewTier: str
    requiresHumanApproval: bool
    blockedReason: str
    estimatedCostUsd: float
    reasons: List[str]


class ToolManifest(Envelope, total=False):
    name: str
    toolKind: str
    version: str
    description: str
    capabilities: List[str]
    permissions: List[str]
    inputSchema: Dict[str, Any]
    outputSchema: Dict[str, Any]
    sandbox: Dict[str, Any]
    endpoint: str


class Workflow(Envelope, total=False):
    name: str
    version: str
    description: str
    budget: Budget
    steps: List[Dict[str, Any]]
