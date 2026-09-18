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
	"sync"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/host"
	"github.com/greadee/aa/toolbox/workflow"
)

// RPCVersion is the supported RPC major.minor.
const RPCVersion = "1.0"

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

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    any    `json:"code"`
	Message string `json:"message"`
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

	mu    sync.Mutex
	cache map[string]any
}

// New returns a service over an invocation host and contract resolver.
func New(h *host.Host, contracts ContractResolver) *Service {
	return &Service{host: h, contracts: contracts, cache: map[string]any{}}
}

// Handle validates the envelope and dispatches one request.
func (s *Service) Handle(ctx context.Context, req Request) Response {
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return errResponse(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
	}
	if !compatible(req.AA) {
		return errResponse(req.ID, CodeIncompatible, "unsupported RPC major")
	}
	if key, ok := idempotencyKey(req); ok {
		s.mu.Lock()
		if cached, found := s.cache[req.Method+"|"+key]; found {
			s.mu.Unlock()
			return Response{JSONRPC: "2.0", ID: req.ID, Result: cached}
		}
		s.mu.Unlock()
	}
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
	if key, ok := idempotencyKey(req); ok {
		s.mu.Lock()
		s.cache[req.Method+"|"+key] = result
		s.mu.Unlock()
	}
	return Response{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func (s *Service) invoke(ctx context.Context, req Request) (any, error) {
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
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: input must be an object", toolbox.ErrInvalid)
		}
		input = m
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

func (s *Service) compileWorkflow(req Request) (any, error) {
	wf, err := workflowParams(req.Params)
	if err != nil {
		return nil, err
	}
	plan, err := workflow.Compile(wf)
	if err != nil {
		return nil, err
	}
	return map[string]any{"plan": plan}, nil
}

func (s *Service) listRegistry(req Request) (any, error) {
	kind := ""
	if req.Params != nil {
		if v, ok := req.Params["toolKind"].(string); ok {
			kind = v
		}
	}
	return map[string]any{"tools": s.host.ListTools(kind)}, nil
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

func compatible(aa *AA) bool {
	if aa == nil || aa.RPCVersion == "" {
		return true
	}
	major := aa.RPCVersion
	for i := 0; i < len(major); i++ {
		if major[i] == '.' {
			major = major[:i]
			break
		}
	}
	return major == "1"
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
	return Response{JSONRPC: "2.0", ID: id, Error: &Error{Code: code, Message: message}}
}

func errResponseFrom(id any, err error) Response {
	switch {
	case errors.Is(err, toolbox.ErrInvalid), errors.Is(err, toolbox.ErrCycle):
		return errResponse(id, CodeInvalidParams, err.Error())
	case errors.Is(err, toolbox.ErrNotFound):
		return errResponse(id, CodeNotFound, err.Error())
	case errors.Is(err, toolbox.ErrDenied), errors.Is(err, toolbox.ErrSandbox), errors.Is(err, toolbox.ErrApproval):
		return errResponse(id, CodeUnauthorized, err.Error())
	case errors.Is(err, toolbox.ErrNoProvider):
		return errResponse(id, CodeUnavailable, err.Error())
	default:
		return errResponse(id, CodeInternalError, err.Error())
	}
}
