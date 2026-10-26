// Package rpc exposes aa-toolbox over the aa inter-module RPC v1 methods.
//
// The adapter is transport-agnostic: callers pass and receive message objects,
// so the kernel can host it over the local socket. Capability checks happen at
// the callee: the adapter resolves the execution contract itself rather than
// trusting the caller.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/host"
	"github.com/greadee/aa/toolbox/workflow"
)

// RPCVersion is the supported RPC major.minor.
const RPCVersion = "1.0"

// supportedContractMajor is the contracts generation this adapter speaks.
const supportedContractMajor = "2"

// Method names from contracts/rpc/rpc-v1.md.
const (
	MethodInvoke          = "toolbox.invoke"
	MethodCompileWorkflow = "toolbox.compileWorkflow"
	MethodRegistry        = "toolbox.registry"
)

// Error codes.
const (
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeIncompatible   = "aa.incompatible"
	CodeUnauthorized   = "aa.unauthorized"
	CodeNotFound       = "aa.not_found"
	CodeConflict       = "aa.conflict"
	CodeApproval       = "aa.approval_required"
	CodeUnavailable    = "aa.unavailable"
)

const (
	defaultTimeout      = 30 * time.Second
	maxExecutionTimeout = 5 * time.Minute
)

// AA is the transport metadata carried on every request.
type AA struct {
	RPCVersion      string `json:"rpcVersion,omitempty"`
	ContractVersion string `json:"contractVersion,omitempty"`
	Caller          string `json:"caller,omitempty"`
	TimeoutMS       int    `json:"timeoutMs,omitempty"`
}

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
	AA      *AA            `json:"aa,omitempty"`
}

// InvokeRequest is the typed payload for toolbox.invoke.
type InvokeRequest struct {
	ToolID              string         `json:"toolId"`
	ExecutionContractID string         `json:"executionContractId"`
	Input               map[string]any `json:"input,omitempty"`
	IdempotencyKey      string         `json:"idempotencyKey,omitempty"`
}

// InvokeOutput is the output envelope returned by toolbox.invoke.
type InvokeOutput struct {
	Output map[string]any `json:"output"`
}

// CompileWorkflowRequest is the typed payload for toolbox.compileWorkflow.
type CompileWorkflowRequest struct {
	Workflow WorkflowParam `json:"workflow"`
}

// WorkflowParam wraps the workflow object (flat or nested).
type WorkflowParam map[string]any

// CompileWorkflowResponse wraps the compiled plan.
type CompileWorkflowResponse struct {
	Plan workflow.Plan `json:"plan"`
}

// ListRegistryRequest is the optional filter for toolbox.registry.
type ListRegistryRequest struct {
	ToolKind string `json:"toolKind,omitempty"`
}

// ListRegistryResponse wraps the returned tool manifests.
type ListRegistryResponse struct {
	Tools []toolbox.Manifest `json:"tools"`
}

// Error is a JSON-RPC 2.0 error object. Governed codes carry data.retryable.
type Error struct {
	Code    any            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// ContractResolver returns the execution contract for an id. The kernel owns
// execution contracts; the adapter resolves them at the callee.
type ContractResolver interface {
	Resolve(id string) (toolbox.ExecutionContract, bool)
}

// Service dispatches toolbox RPC methods.
type Service struct {
	host      *host.Host
	contracts ContractResolver
	metrics   toolbox.Metrics

	mu       sync.Mutex
	inflight map[string]*cacheResult
}

// cacheResult holds the result of an in-flight request for deduplication.
type cacheResult struct {
	respCh      chan struct{} // closed when the first caller finishes
	resp        Response      // result set by the first caller
	fingerprint string        // canonical request identity for conflict detection
}

// Metrics and MetricsSnapshot are retained as source-compatible aliases for
// callers that used the former rpc-local metrics type. Use toolbox.MemMetrics
// for new code.
type Metrics = toolbox.MemMetrics
type MetricsSnapshot = toolbox.MetricsSnapshot

// New returns a service over an invocation host and contract resolver.
// Optionally pass a Metrics implementation; the default is a no-op collector.
func New(h *host.Host, contracts ContractResolver, metrics ...toolbox.Metrics) *Service {
	m := toolbox.Metrics(toolbox.NoopMetrics{})
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	}
	return &Service{host: h, contracts: contracts, metrics: m, inflight: map[string]*cacheResult{}}
}

// Capabilities reports the RPC/contract versions and methods this adapter speaks,
// so a host can negotiate before calling.
func (s *Service) Capabilities() Capabilities {
	return Capabilities{
		RPCVersion:      RPCVersion,
		ContractVersion: supportedContractMajor + ".0",
		Methods:         []string{MethodInvoke, MethodCompileWorkflow, MethodRegistry},
	}
}

// Metrics returns a snapshot when the service is using MemMetrics. Other
// implementations return a zero snapshot; callers can also hold the Metrics
// value they passed to New.
func (s *Service) Metrics() toolbox.MetricsSnapshot {
	if m, ok := s.metrics.(*toolbox.MemMetrics); ok {
		return m.Snapshot()
	}
	return toolbox.MetricsSnapshot{}
}

// Capabilities is the advertised version and method set of an adapter.
type Capabilities struct {
	RPCVersion      string   `json:"rpcVersion"`
	ContractVersion string   `json:"contractVersion"`
	Methods         []string `json:"capabilities"`
}

// Handle validates the envelope and dispatches one request.
func (s *Service) Handle(ctx context.Context, req Request) Response {
	start := time.Now()
	resp := s.doHandle(ctx, req)
	s.metrics.RecordRPC(req.Method, time.Since(start), resp.Error == nil)
	if resp.Error != nil {
		s.metrics.RecordRPCDenial(req.Method, errorCategory(resp.Error.Data))
	}
	return resp
}

// doHandle is the internal handle logic (without metrics recording).
func (s *Service) doHandle(ctx context.Context, req Request) Response {
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return errResponseData(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"", errorData("invalid", false, "", ""))
	}
	if !compatible(req.AA) {
		return errResponseData(req.ID, CodeIncompatible, "unsupported RPC or contract major", errorData("incompatible", false, "", ""))
	}
	// AA is optional on the wire for compatibility, but every request is
	// bounded. An explicit positive TimeoutMS overrides the server default.
	timeout := defaultTimeout
	if req.AA != nil && req.AA.TimeoutMS > 0 {
		timeout = time.Duration(req.AA.TimeoutMS) * time.Millisecond
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	if key, ok := idempotencyKey(req); ok {
		return s.handleWithDedup(ctx, req, key)
	}
	return s.handleSingle(ctx, req)
}

// handleWithDedup guarantees one execution per method/idempotency-key. It
// deduplicates only in-flight requests. Completed replays are delegated to the
// host, whose cache is tool/contract scoped and re-evaluates policy first.
func (s *Service) handleWithDedup(ctx context.Context, req Request, key string) Response {
	dedupKey := req.Method + "|" + key
	fingerprint := requestFingerprint(req)

	s.mu.Lock()
	if cr, exists := s.inflight[dedupKey]; exists {
		if cr.fingerprint != fingerprint {
			s.mu.Unlock()
			toolID, contractID := requestErrorIDs(req)
			return errResponseData(req.ID, CodeConflict, "idempotency key reused with a different request",
				errorData("conflict", false, toolID, contractID))
		}
		s.mu.Unlock()
		return waitForResponse(ctx, req, cr)
	}
	cr := &cacheResult{respCh: make(chan struct{}), fingerprint: fingerprint}
	s.inflight[dedupKey] = cr
	s.mu.Unlock()
	s.metrics.RecordCacheMiss(dedupKey)

	// The provider's lifetime is not tied to any single waiter's deadline. Each
	// caller still enforces its own deadline while waiting; execution keeps its
	// own bounded context so a short-deadline first caller cannot cancel work a
	// longer-deadline waiter is still entitled to receive.
	execCtx, execCancel := context.WithTimeout(context.WithoutCancel(ctx), maxExecutionTimeout)
	go func() {
		defer execCancel()
		resp := s.dispatch(execCtx, req)
		if execCtx.Err() != nil {
			resp = errResponseFrom(req, execCtx.Err())
		}
		s.mu.Lock()
		cr.resp = resp
		delete(s.inflight, dedupKey)
		close(cr.respCh)
		s.mu.Unlock()
	}()
	return waitForResponse(ctx, req, cr)
}

func waitForResponse(ctx context.Context, req Request, cr *cacheResult) Response {
	select {
	case <-cr.respCh:
		resp := cr.resp
		resp.ID = req.ID
		return resp
	case <-ctx.Done():
		return errResponseFrom(req, ctx.Err())
	}
}

func requestFingerprint(req Request) string {
	payload, _ := json.Marshal(struct {
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
		Caller string         `json:"caller,omitempty"`
	}{
		Method: req.Method,
		Params: req.Params,
		Caller: func() string {
			if req.AA == nil {
				return ""
			}
			return req.AA.Caller
		}(),
	})
	return toolbox.Hash(payload)
}

func errorCategory(data map[string]any) string {
	if data == nil {
		return "internal"
	}
	if category, ok := data["category"].(string); ok {
		return category
	}
	return "internal"
}

// handleSingle processes a request without an idempotency key.
func (s *Service) handleSingle(ctx context.Context, req Request) Response {
	return s.dispatch(ctx, req)
}

// dispatch executes the requested method and returns the response.
func (s *Service) dispatch(ctx context.Context, req Request) Response {
	var (
		result any
		err    error
	)
	switch req.Method {
	case MethodInvoke:
		result, err = s.invoke(ctx, req)
	case MethodCompileWorkflow:
		result, err = s.compileWorkflow(req)
	case MethodRegistry:
		result, err = s.listRegistry(req)
	default:
		return errResponseData(req.ID, CodeMethodNotFound, fmt.Sprintf("unknown method %q", req.Method), errorData("method_not_found", false, "", ""))
	}
	if err != nil {
		return errResponseFrom(req, err)
	}
	return Response{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func (s *Service) invoke(ctx context.Context, req Request) (any, error) {
	var typed InvokeRequest
	if err := json.Unmarshal(mapToJSON(req.Params), &typed); err != nil {
		// fall back to manual parsing for backward compatibility.
		toolID, err := stringField(req.Params, "toolId")
		if err != nil {
			return nil, err
		}
		contractID := ""
		if v, ok := req.Params["executionContractId"].(string); ok {
			contractID = v
		}
		if contractID == "" {
			return nil, fmt.Errorf("%w: executionContractId is required", toolbox.ErrInvalid)
		}
		if s.contracts == nil {
			return nil, fmt.Errorf("%w: no execution contract for %q", toolbox.ErrDenied, contractID)
		}
		contract, ok := s.contracts.Resolve(contractID)
		if !ok {
			return nil, fmt.Errorf("%w: execution contract %q", toolbox.ErrNotFound, contractID)
		}
		input := map[string]any{}
		if raw, ok := req.Params["input"]; ok && raw != nil {
			if m, ok := raw.(map[string]any); ok {
				input = m
			}
		}
		idem := ""
		if v, ok := req.Params["idempotencyKey"].(string); ok {
			idem = v
		}
		result, err := s.host.Invoke(ctx, toolbox.Invocation{
			ToolID:              toolbox.ToolID(toolID),
			Input:               input,
			ExecutionContractID: contractID,
			IdempotencyKey:      idem,
		}, contract)
		if err != nil {
			return nil, err
		}
		return map[string]any{"output": result.Output}, nil
	}
	if s.contracts == nil {
		return nil, fmt.Errorf("%w: no execution contract for %q", toolbox.ErrDenied, typed.ExecutionContractID)
	}
	contract, ok := s.contracts.Resolve(typed.ExecutionContractID)
	if !ok {
		return nil, fmt.Errorf("%w: execution contract %q", toolbox.ErrNotFound, typed.ExecutionContractID)
	}
	result, err := s.host.Invoke(ctx, toolbox.Invocation{
		ToolID:              toolbox.ToolID(typed.ToolID),
		Input:               typed.Input,
		ExecutionContractID: typed.ExecutionContractID,
		IdempotencyKey:      typed.IdempotencyKey,
	}, contract)
	if err != nil {
		return nil, err
	}
	return InvokeOutput{Output: result.Output}, nil
}

func (s *Service) compileWorkflow(req Request) (any, error) {
	var typed CompileWorkflowRequest
	if req.Params != nil {
		if err := json.Unmarshal(mapToJSON(req.Params), &typed); err != nil {
			// fall back to manual parsing for backward compatibility.
			wf, err := workflowParams(req.Params)
			if err != nil {
				return nil, err
			}
			plan, err := workflow.Compile(wf)
			if err != nil {
				return nil, err
			}
			return CompileWorkflowResponse{Plan: plan}, nil
		}
	}
	wf, err := workflowParams(map[string]any(typed.Workflow))
	if err != nil {
		return nil, err
	}
	plan, err := workflow.Compile(wf)
	if err != nil {
		return nil, err
	}
	return CompileWorkflowResponse{Plan: plan}, nil
}

func (s *Service) listRegistry(req Request) (any, error) {
	var typed ListRegistryRequest
	if req.Params != nil {
		if err := json.Unmarshal(mapToJSON(req.Params), &typed); err != nil {
			// fall back to manual parsing.
			kind := ""
			if v, ok := req.Params["toolKind"].(string); ok {
				kind = v
			}
			return ListRegistryResponse{Tools: s.host.ListTools(kind)}, nil
		}
	}
	return ListRegistryResponse{Tools: s.host.ListTools(typed.ToolKind)}, nil
}

func mapToJSON(m map[string]any) []byte {
	b, _ := json.Marshal(m)
	return b
}

func workflowParams(params map[string]any) (toolbox.Workflow, error) {
	raw := any(params)
	if inner, ok := params["workflow"]; ok {
		raw = inner
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return toolbox.Workflow{}, fmt.Errorf("%w: workflow: %v", toolbox.ErrInvalid, err)
	}
	var wf toolbox.Workflow
	if err := json.Unmarshal(encoded, &wf); err != nil {
		return toolbox.Workflow{}, fmt.Errorf("%w: workflow: %v", toolbox.ErrInvalid, err)
	}
	return wf, nil
}

// compatible reports whether the request's envelope and contract majors are
// supported. An absent version is treated as compatible; a present unsupported
// major fails closed (aa.incompatible).
func compatible(aa *AA) bool {
	if aa == nil {
		return true
	}
	if aa.RPCVersion != "" && major(aa.RPCVersion) != "1" {
		return false
	}
	if aa.ContractVersion != "" && major(aa.ContractVersion) != supportedContractMajor {
		return false
	}
	return true
}

func major(version string) string {
	if i := strings.IndexByte(version, '.'); i >= 0 {
		return version[:i]
	}
	return version
}

func idempotencyKey(req Request) (string, bool) {
	if req.Params == nil {
		return "", false
	}
	value, ok := req.Params["idempotencyKey"].(string)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func stringField(params map[string]any, field string) (string, error) {
	value, ok := params[field].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("%w: %s is required", toolbox.ErrInvalid, field)
	}
	return value, nil
}

func errResponse(id any, code any, message string) Response {
	return errResponseData(id, code, message, nil)
}

func errResponseData(id any, code any, message string, data map[string]any) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &Error{Code: code, Message: message, Data: data}}
}

func errorData(category string, retryableValue bool, toolID, contractID string) map[string]any {
	data := map[string]any{
		"category":  category,
		"retryable": retryableValue,
	}
	if toolID != "" {
		data["toolId"] = toolID
	}
	if contractID != "" {
		data["executionContractId"] = contractID
	}
	return data
}

func requestErrorIDs(req Request) (toolID, contractID string) {
	if req.Params == nil {
		return "", ""
	}
	toolID, _ = req.Params["toolId"].(string)
	contractID, _ = req.Params["executionContractId"].(string)
	return toolID, contractID
}

func toolboxErrorName(err error) string {
	switch {
	case errors.Is(err, toolbox.ErrInvalid):
		return "ErrInvalid"
	case errors.Is(err, toolbox.ErrNotFound):
		return "ErrNotFound"
	case errors.Is(err, toolbox.ErrConflict):
		return "ErrConflict"
	case errors.Is(err, toolbox.ErrDenied):
		return "ErrDenied"
	case errors.Is(err, toolbox.ErrSandbox):
		return "ErrSandbox"
	case errors.Is(err, toolbox.ErrNoProvider):
		return "ErrNoProvider"
	case errors.Is(err, toolbox.ErrCycle):
		return "ErrCycle"
	case errors.Is(err, toolbox.ErrFailed):
		return "ErrFailed"
	case errors.Is(err, toolbox.ErrUnsupported):
		return "ErrUnsupported"
	case errors.Is(err, toolbox.ErrApproval):
		return "ErrApproval"
	default:
		return ""
	}
}

func withToolboxError(data map[string]any, err error) map[string]any {
	if name := toolboxErrorName(err); name != "" {
		data["toolboxError"] = name
	}
	return data
}

func errResponseFrom(req Request, err error) Response {
	toolID, contractID := requestErrorIDs(req)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return errResponseData(req.ID, CodeUnavailable, "request timed out",
			withToolboxError(errorData("timeout", true, toolID, contractID), err))
	case errors.Is(err, context.Canceled):
		return errResponseData(req.ID, CodeUnavailable, "request canceled",
			withToolboxError(errorData("canceled", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrInvalid), errors.Is(err, toolbox.ErrCycle):
		return errResponseData(req.ID, CodeInvalidParams, err.Error(),
			withToolboxError(errorData("invalid", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrNotFound):
		return errResponseData(req.ID, CodeNotFound, err.Error(),
			withToolboxError(errorData("not_found", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrConflict):
		return errResponseData(req.ID, CodeConflict, err.Error(),
			withToolboxError(errorData("conflict", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrDenied), errors.Is(err, toolbox.ErrSandbox):
		return errResponseData(req.ID, CodeUnauthorized, err.Error(),
			withToolboxError(errorData("unauthorized", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrApproval):
		return errResponseData(req.ID, CodeApproval, err.Error(),
			withToolboxError(errorData("approval_required", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrUnsupported):
		return errResponseData(req.ID, CodeUnavailable, err.Error(),
			withToolboxError(errorData("unsupported", false, toolID, contractID), err))
	case errors.Is(err, toolbox.ErrNoProvider):
		return errResponseData(req.ID, CodeUnavailable, err.Error(),
			withToolboxError(errorData("unavailable", true, toolID, contractID), err))
	default:
		return errResponseData(req.ID, CodeInternalError, err.Error(),
			withToolboxError(errorData("internal", false, toolID, contractID), err))
	}
}
