// TypeScript bindings for the aa contract schemas v1.
// The JSON Schema files under contracts/schemas/v1 are the source of truth.
// Objects are open: unknown properties are permitted and ignored.

export const CONTRACT_VERSION = "1.0" as const;

export type ContractVersion = string;
export type Identifier = string;
export type Timestamp = string;

export interface Reference {
  kind: string;
  id: Identifier;
  version?: string;
}

export type ActorKind = "human" | "agent" | "system" | "service";

export interface Actor {
  kind: ActorKind;
  id: Identifier;
  role?: string;
  trade?: string;
  workerId?: Identifier;
  model?: string;
}

export type Confidence = "exact" | "correlated" | "observed" | "inferred";

export interface Provenance {
  source: string;
  producedAt: Timestamp;
  producedBy?: Actor;
  confidence?: Confidence;
  evidence?: Reference[];
  contentHash?: string;
}

export interface Envelope {
  contractVersion: ContractVersion;
  kind: string;
  id: Identifier;
  projectId?: Identifier;
  createdAt?: Timestamp;
  revision?: number;
  provenance?: Provenance;
}

export interface Budget {
  maxCalls?: number;
  maxTokens?: number;
  maxCostUsd?: number;
  maxDurationSeconds?: number;
}

export type ProjectState =
  | "CREATED" | "DISCOVERY" | "PLANNING" | "EXECUTING" | "VERIFYING"
  | "COMPLETE" | "BLOCKED" | "FAILED" | "CANCELLED" | "PAUSED";

export type WorkPackageState =
  | "PROPOSED" | "READY" | "QUEUED" | "RUNNING"
  | "COMPLETE" | "VERIFYING" | "VERIFIED" | "DEFICIENT";

export type AssignmentState =
  | "planned" | "leased" | "preparing" | "running" | "paused"
  | "collecting" | "awaiting_gates" | "accepted" | "failed" | "canceled" | "expired";

export type ReadinessState = "blocked" | "ready" | "dispatched" | "satisfied";

export type IssueState = "open" | "in_progress" | "review" | "closed" | "deferred" | "cancelled";

export type MemoryLifecycle =
  | "EPHEMERAL" | "CANDIDATE" | "VALIDATED" | "ACTIVE" | "SUPERSEDED" | "ARCHIVED";

export type EventType =
  | "PROJECT_REGISTERED" | "PROJECT_UPDATED"
  | "WORK_PACKAGE_CREATED" | "WORK_PACKAGE_READY" | "WORK_PACKAGE_COMPLETED"
  | "DEPENDENCY_GRAPH_REVISED"
  | "EXECUTION_PLANNED" | "EXECUTION_LEASED" | "EXECUTION_STARTED" | "EXECUTION_PAUSED"
  | "EXECUTION_RESUMED" | "EXECUTION_COMPLETED" | "EXECUTION_FAILED" | "EXECUTION_EXPIRED"
  | "RESULT_RECORDED" | "TEST_RECORDED" | "REVIEW_RECORDED"
  | "HANDOFF_CREATED" | "ARTIFACT_RECORDED"
  | "WORK_ACCEPTED" | "WORK_REJECTED" | "DEFICIENCY_RAISED"
  | "TELEMETRY_RECORDED"
  | "MEMORY_CANDIDATE_CREATED" | "MEMORY_VALIDATED" | "MEMORY_PROMOTED"
  | "MEMORY_SUPERSEDED" | "MEMORY_ARCHIVED";

export interface EventAggregate {
  kind: "project" | "work_package" | "assignment" | "attempt" | "artifact"
    | "issue" | "strategy" | "memory" | "tool" | "workflow";
  id: Identifier;
  version?: number;
}

export interface Event extends Envelope {
  kind: "event";
  sequence: number;
  occurredAt: Timestamp;
  recordedAt?: Timestamp;
  type: EventType;
  aggregate: EventAggregate;
  actor: Actor;
  causationId?: Identifier;
  correlationId?: Identifier;
  payload?: Record<string, unknown>;
}

export interface Scope {
  allowed?: string[];
  inspect?: string[];
  forbidden?: string[];
}

export interface WorkPackage extends Envelope {
  kind: "work_package";
  title: string;
  description?: string;
  state: WorkPackageState;
  trade?: string;
  role?: string;
  dependencies?: Identifier[];
  inputs?: Reference[];
  deliverables?: string[];
  acceptance: string[];
  scope?: Scope;
  budget?: Budget;
  risk?: "low" | "moderate" | "high" | "critical";
}

export interface TaskGraphNode {
  workPackageId: Identifier;
  dependsOn?: Identifier[];
  readiness: ReadinessState;
}

export interface TaskGraph extends Envelope {
  kind: "task_graph";
  projectId: Identifier;
  revision: number;
  nodes: TaskGraphNode[];
  digest?: string;
}

export type Capability =
  | "read_project" | "write_workspace" | "execute_command" | "run_tests"
  | "network_access" | "install_dependencies" | "call_model" | "read_secrets"
  | "create_artifact" | "open_pull_request" | "merge" | "deploy";

export interface RuntimeSpec {
  kind: string;
  provider?: string;
  model?: string;
  nodeId?: Identifier;
}

export interface WorkspaceSpec {
  required?: boolean;
  kind?: "none" | "git_worktree" | "sandbox";
  root?: string;
}

export interface ExecutionContract extends Envelope {
  kind: "execution_contract";
  workPackageId: Identifier;
  assignmentId: Identifier;
  capabilities: Capability[];
  denied?: Capability[];
  runtime?: RuntimeSpec;
  workspace?: WorkspaceSpec;
  budget: Budget;
  gates?: string[];
  expiresAt?: Timestamp;
  digest?: string;
}

export interface ResultArtifact {
  kind: string;
  id: Identifier;
  hash?: string;
  uri?: string;
}

export interface TestResult {
  name: string;
  outcome: "passed" | "failed" | "skipped" | "error";
  durationMs?: number;
}

export interface Failure {
  class?: string;
  message?: string;
  retryable?: boolean;
}

export interface ResultEnvelope extends Envelope {
  kind: "result_envelope";
  attemptId: Identifier;
  assignmentId: Identifier;
  workPackageId: Identifier;
  status: "succeeded" | "failed" | "partial" | "blocked" | "cancelled";
  summary?: string;
  artifacts?: ResultArtifact[];
  tests?: TestResult[];
  telemetryRef?: Reference;
  failure?: Failure;
  producedBy?: Actor;
}

export interface CallCounts {
  total?: number;
  local?: number;
  cloud?: number;
  retries?: number;
}

export interface TokenCounts {
  input?: number;
  output?: number;
}

export interface Telemetry extends Envelope {
  kind: "telemetry";
  attemptId: Identifier;
  assignmentId?: Identifier;
  metricVersion: string;
  outcome: "succeeded" | "failed" | "partial" | "blocked" | "cancelled" | "unknown";
  calls?: CallCounts;
  tokens?: TokenCounts;
  costUsd?: number;
  durationMs?: number;
  resources?: Record<string, unknown>;
  versions?: Record<string, string>;
}

export type TracePhase =
  | "observe" | "plan" | "model" | "tool" | "edit"
  | "test" | "gate" | "verify" | "integrate";

export type TraceOutcome = "succeeded" | "failed" | "skipped" | "blocked";

export interface TraceStep {
  sequence: number;
  at?: Timestamp;
  phase: TracePhase;
  actor?: Actor;
  operation?: string;
  target?: Reference;
  inputHash?: string;
  outputHash?: string;
  outcome: TraceOutcome;
  errorClass?: string;
  durationMs?: number;
  tokens?: TokenCounts;
  costUsd?: number;
  redacted?: boolean;
  evidence?: Reference[];
}

export interface Trace extends Envelope {
  kind: "trace";
  attemptId: Identifier;
  assignmentId?: Identifier;
  workPackageId?: Identifier;
  redactionVersion?: string;
  truncated?: boolean;
  steps: TraceStep[];
}

export interface ProjectRecord extends Envelope {
  kind: "project_record";
  name: string;
  state: ProjectState;
  layoutVersion: string;
  authorityId?: Identifier;
  replicaIds?: Identifier[];
  records?: Reference[];
  digest?: string;
}

export interface MemoryContent {
  summary: string;
  details?: string;
  tags?: string[];
}

export interface Applicability {
  roles?: string[];
  trades?: string[];
  languages?: string[];
}

export interface MemoryRecord extends Envelope {
  kind: "memory_record";
  level: "session" | "task" | "project" | "role" | "workforce";
  lifecycle: MemoryLifecycle;
  title?: string;
  content: MemoryContent;
  applicability?: Applicability;
  evidence?: Reference[];
  confidence?: number;
  supersedes?: Identifier;
}

export interface IssueLinks {
  workPackages?: Identifier[];
  commits?: string[];
  pullRequests?: Reference[];
  adrs?: string[];
}

export interface Issue extends Envelope {
  kind: "issue";
  title: string;
  type: "feature" | "user_story" | "bug" | "tech_debt" | "refactor"
    | "test" | "documentation" | "audit" | "investigation" | "deficiency";
  status: IssueState;
  severity?: "P0" | "P1" | "P2" | "P3" | "none";
  body?: string;
  milestone?: string;
  forgeRef?: Reference;
  links?: IssueLinks;
  deferredTo?: Identifier;
}

export interface Strategy extends Envelope {
  kind: "strategy";
  title: string;
  lifecycle: MemoryLifecycle;
  guidance: string;
  whenToUse?: string;
  whenNotToUse?: string;
  applicability?: Applicability;
  evidence?: Reference[];
  supersedes?: Identifier;
}

export type RoutingPolicy =
  | "local_only" | "cloud_only" | "local_first"
  | "expert_first" | "adaptive" | "budget_constrained";

export interface RouteRequest extends Envelope {
  kind: "route_request";
  attemptId?: Identifier;
  prompt: string;
  context?: string;
  policy?: RoutingPolicy;
  budget?: Budget;
  metadata?: Record<string, unknown>;
}

export interface RouteResponse extends Envelope {
  kind: "route_response";
  requestId: Identifier;
  route: "local" | "hybrid" | "cloud" | "blocked";
  decisionLevel: "routine" | "significant" | "major" | "critical";
  plannerTier?: "local" | "expert" | "none";
  executorTier?: "local" | "expert" | "none";
  reviewTier?: "local" | "expert" | "none";
  requiresHumanApproval: boolean;
  blockedReason?: string;
  estimatedCostUsd?: number;
  reasons?: string[];
}

export interface SandboxSpec {
  required?: boolean;
  kind?: "none" | "process" | "container" | "os";
  network?: boolean;
}

export interface ToolManifest extends Envelope {
  kind: "tool_manifest";
  name: string;
  toolKind: "tool" | "plugin" | "mcp" | "builtin";
  version: string;
  description?: string;
  capabilities: string[];
  permissions?: string[];
  inputSchema?: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;
  sandbox?: SandboxSpec;
  endpoint?: string;
}

export interface WorkflowStep {
  id: Identifier;
  kind: "task" | "tool" | "gate" | "human_approval" | "parallel" | "conditional";
  role?: string;
  trade?: string;
  toolId?: Identifier;
  dependsOn?: Identifier[];
  retries?: number;
  gates?: string[];
  onFailure?: "fail" | "retry" | "skip" | "compensate";
  config?: Record<string, unknown>;
}

export interface Workflow extends Envelope {
  kind: "workflow";
  name?: string;
  version: string;
  description?: string;
  budget?: Budget;
  steps: WorkflowStep[];
}

export type ContractObject =
  | Event | WorkPackage | TaskGraph | ExecutionContract | ResultEnvelope
  | Telemetry | Trace | ProjectRecord | MemoryRecord | Issue | Strategy
  | RouteRequest | RouteResponse | ToolManifest | Workflow;

export function isContractObject(value: unknown): value is ContractObject {
  return typeof value === "object" && value !== null && "kind" in value && "id" in value;
}
