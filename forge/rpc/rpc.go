// Package rpc exposes the forge over the aa inter-module RPC v1 methods.
//
// The adapter is transport-agnostic: callers pass and receive message objects,
// so the kernel can host it over the local socket. Merging is deliberately not
// exposed here; it stays on the control-plane path behind the human gate.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/greadee/aa/forge"
)

// RPCVersion is the supported RPC major.minor.
const RPCVersion = "1.0"

// Method names from contracts/rpc/rpc-v1.md.
const (
	MethodCreateIssue = "forge.createIssue"
	MethodOpenPR      = "forge.openPullRequest"
	MethodCheckpoint  = "forge.checkpoint"
	MethodRelease     = "forge.release"
)

// Error codes.
const (
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeIncompatible   = "aa.incompatible"
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

// Forge is the capability set the adapter needs.
type Forge interface {
	forge.Issues
	forge.PullRequests
	forge.Checkpoints
	forge.Releases
}

// Service dispatches forge RPC methods.
type Service struct {
	forge Forge
	mu    sync.Mutex
	cache map[string]any
}

// New returns a service over f.
func New(f Forge) *Service {
	return &Service{forge: f, cache: map[string]any{}}
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
	case MethodCreateIssue:
		result, err = s.createIssue(ctx, req)
	case MethodOpenPR:
		result, err = s.openPullRequest(ctx, req)
	case MethodCheckpoint:
		result, err = s.checkpoint(ctx, req)
	case MethodRelease:
		result, err = s.release(ctx, req)
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

func (s *Service) createIssue(ctx context.Context, req Request) (any, error) {
	repoID, err := stringField(req.Params, "repoId")
	if err != nil {
		return nil, err
	}
	var issue forge.Issue
	if err := decodeField(req.Params, "issue", &issue); err != nil {
		return nil, err
	}
	created, err := s.forge.CreateIssue(ctx, repoID, issue)
	if err != nil {
		return nil, err
	}
	return refResult(created.Ref), nil
}

func (s *Service) openPullRequest(ctx context.Context, req Request) (any, error) {
	repoID, err := stringField(req.Params, "repoId")
	if err != nil {
		return nil, err
	}
	var pr forge.PullRequest
	if err := decodeField(req.Params, "pullRequest", &pr); err != nil {
		return nil, err
	}
	created, err := s.forge.OpenPullRequest(ctx, repoID, pr)
	if err != nil {
		return nil, err
	}
	return refResult(created.Ref), nil
}

func (s *Service) checkpoint(ctx context.Context, req Request) (any, error) {
	repoID, err := stringField(req.Params, "repoId")
	if err != nil {
		return nil, err
	}
	var checkpoint forge.Checkpoint
	if err := decodeField(req.Params, "checkpoint", &checkpoint); err != nil {
		return nil, err
	}
	created, err := s.forge.CreateCheckpoint(ctx, repoID, checkpoint)
	if err != nil {
		return nil, err
	}
	return refResult(forge.Ref{Kind: "checkpoint", ID: created.ID}), nil
}

func (s *Service) release(ctx context.Context, req Request) (any, error) {
	repoID, err := stringField(req.Params, "repoId")
	if err != nil {
		return nil, err
	}
	var release forge.Release
	if err := decodeField(req.Params, "release", &release); err != nil {
		return nil, err
	}
	created, err := s.forge.CreateRelease(ctx, repoID, release)
	if err != nil {
		return nil, err
	}
	return refResult(forge.Ref{Kind: "release", ID: created.ID}), nil
}

func refResult(ref forge.Ref) map[string]any {
	return map[string]any{"forgeRef": map[string]any{"kind": ref.Kind, "id": ref.ID, "url": ref.URL}}
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
		return "", fmt.Errorf("%w: %s is required", forge.ErrInvalid, field)
	}
	return value, nil
}

func decodeField(params map[string]any, field string, target any) error {
	value, ok := params[field]
	if !ok {
		return fmt.Errorf("%w: %s is required", forge.ErrInvalid, field)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", forge.ErrInvalid, field, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%w: %s: %v", forge.ErrInvalid, field, err)
	}
	return nil
}

func errResponse(id any, code any, message string) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &Error{Code: code, Message: message}}
}

func errResponseFrom(id any, err error) Response {
	switch {
	case errors.Is(err, forge.ErrInvalid), errors.Is(err, forge.ErrNotFound):
		return errResponse(id, CodeInvalidParams, err.Error())
	case errors.Is(err, forge.ErrRateLimited):
		return errResponse(id, CodeUnavailable, err.Error())
	default:
		return errResponse(id, CodeInternalError, err.Error())
	}
}
