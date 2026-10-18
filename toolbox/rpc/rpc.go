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
	CodeUnavailable    = "aa.unavailable"
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

	mu       sync.Mutex
	cache    map[string]any
	inflight map[string]*cacheResult
}

// cacheResult holds the result of an in-flight request for deduplication.
type cacheResult struct {
	respCh chan struct{} // closed when the first caller finishes
	resp   Response      // result set by the first caller
}

// New returns a service over an invocation host and contract resolver.
func New(h *host.Host, contracts ContractResolver) *Service {
	return &Service{host: h, contracts: contracts, cache: map[string]any{}, inflight: map[string]*cacheResult{}}
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

// Capabilities is the advertised version and method set of an adapter.
type Capabilities struct {
	RPCVersion      string   `json:"rpcVersion"`
	ContractVersion string   `json:"contractVersion"`
	Methods         []string `json:"capabilities"`
}

// Handle validates the envelope and dispatches one request.
func (s *Service) Handle(ctx context.Context, req Request) Response {
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return errResponse(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
	}
	if !compatible(req.AA) {
		return errResponseData(req.ID, CodeIncompatible, "unsupported RPC or contract major", retryable(false))
	}
	// apply timeout from aa envelope if specified.
	if req.AA != nil && req.AA.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.AA.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	if key, ok := idempotencyKey(req); ok {
		return s.handleWithDedup(ctx, req, key)
	}
	return s.handleSingle(ctx, req)
}

// handleWithDedup deduplicates in-flight requests sharing the same idempotency
// key: only the first caller executes the method; others wait and reuse the result.
func (s *Service) handleWithDedup(ctx context.Context, req Request, key string) Response {
	s.mu.Lock()
	if cr, exists := s.inflight[req.Method+"|"+key]; exists {
		// another caller is already executing this request; wait for its result.
		ch := cr.respCh
		s.mu.Unlock()
		<-ch // blocks until the first caller closes it
		s.mu.Lock()
		resp := cr.resp
		s.cache[req.Method+"|"+key] = resp.Result
		delete(s.inflight, req.Method+"|"+key)
		s.mu.Unlock()
		return resp
	}
	// first caller: create a result holder and proceed to execute.
	cr := &cacheResult{respCh: make(chan struct{})}
	s.inflight[req.Method+"|"+key] = cr
	s.mu.Unlock()

	// execute the method outside the lock.
	resp := s.dispatch(ctx, req)

	// signal all waiters and clean up.
	s.mu.Lock()
	cr.resp = resp
	s.cache[req.Method+"|"+key] = resp.Result
	delete(s.inflight, req.Method+"|"+key)
	close(cr.respCh) // unblocks all waiters
	s.mu.Unlock()
	return resp
}

// handleSingle processes a non-idempotent request.
func (s *Service) handleSingle(ctx context.Context, req Request) Response {
	s.mu.Lock()
	if key, ok := idempotencyKey(req); ok {
		s.mu.Unlock()
		// idempotency key present but not used for dedup—still cache the result.
		result := s.dispatch(ctx, req)
		s.mu.Lock()
		s.cache[req.Method+"|"+key] = result.Result
		s.mu.Unlock()
		return result
	}
	s.mu.Unlock()
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
		return errResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("unknown method %q", req.Method))
	}
	if err != nil {
		return errResponseFrom(req.ID, err)
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

func retryable(v bool) map[string]any { return map[string]any{"retryable": v} }

func errResponseFrom(id any, err error) Response {
	switch {
	case errors.Is(err, toolbox.ErrInvalid), errors.Is(err, toolbox.ErrCycle):
		return errResponseData(id, CodeInvalidParams, err.Error(), retryable(false))
	case errors.Is(err, toolbox.ErrNotFound):
		return errResponseData(id, CodeNotFound, err.Error(), retryable(false))
	case errors.Is(err, toolbox.ErrDenied), errors.Is(err, toolbox.ErrSandbox), errors.Is(err, toolbox.ErrApproval):
		return errResponseData(id, CodeUnauthorized, err.Error(), retryable(false))
	case errors.Is(err, toolbox.ErrNoProvider):
		return errResponseData(id, CodeUnavailable, err.Error(), retryable(true))
	default:
		return errResponseData(id, CodeInternalError, err.Error(), retryable(false))
	}
}
