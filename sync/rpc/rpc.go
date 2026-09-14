// Package rpc exposes aa-sync over the aa inter-module RPC v1 methods.
//
// The adapter is transport-agnostic: callers pass and receive message objects,
// so the kernel can host it over the local socket. It carries no execution
// surface; it only distributes work and artifacts and reports status.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/distribute"
)

// RPCVersion is the supported RPC major.minor.
const RPCVersion = "1.0"

// Method names from contracts/rpc/rpc-v1.md.
const (
	MethodDistribute    = "sync.distribute"
	MethodFetchArtifact = "sync.fetchArtifact"
	MethodStatus        = "sync.status"
)

// Error codes.
const (
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeIncompatible   = "aa.incompatible"
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

// Service dispatches sync RPC methods.
type Service struct {
	distributor *distribute.Distributor
	mu          sync.Mutex
	cache       map[string]any
}

// New returns a service over a distributor.
func New(d *distribute.Distributor) *Service {
	return &Service{distributor: d, cache: map[string]any{}}
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
	case MethodDistribute:
		result, err = s.distribute(ctx, req)
	case MethodFetchArtifact:
		result, err = s.fetchArtifact(ctx, req)
	case MethodStatus:
		result, err = s.status(req)
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

func (s *Service) distribute(ctx context.Context, req Request) (any, error) {
	node, err := stringField(req.Params, "nodeId")
	if err != nil {
		return nil, err
	}
	wp, err := s.workPackage(req.Params)
	if err != nil {
		return nil, err
	}
	receipt, err := s.distributor.Distribute(ctx, wp, aasync.NodeID(node))
	if err != nil {
		return nil, err
	}
	return map[string]any{"accepted": true, "transferId": string(receipt.TransferID)}, nil
}

func (s *Service) fetchArtifact(ctx context.Context, req Request) (any, error) {
	node, err := stringField(req.Params, "nodeId")
	if err != nil {
		return nil, err
	}
	artifactID, err := stringField(req.Params, "artifactId")
	if err != nil {
		return nil, err
	}
	receipt, err := s.distributor.FetchArtifact(ctx, artifactID, aasync.NodeID(node))
	if err != nil {
		return nil, err
	}
	return map[string]any{"transferId": string(receipt.TransferID), "hash": receipt.Hash}, nil
}

func (s *Service) status(req Request) (any, error) {
	node := ""
	if req.Params != nil {
		if v, ok := req.Params["nodeId"].(string); ok {
			node = v
		}
	}
	receipts := s.distributor.Status(aasync.NodeID(node))
	return map[string]any{"transfers": receipts}, nil
}

func (s *Service) workPackage(params map[string]any) (aasync.WorkPackage, error) {
	if raw, ok := params["workPackage"]; ok {
		var wp aasync.WorkPackage
		encoded, err := json.Marshal(raw)
		if err != nil {
			return aasync.WorkPackage{}, fmt.Errorf("%w: workPackage: %v", aasync.ErrInvalid, err)
		}
		if err := json.Unmarshal(encoded, &wp); err != nil {
			return aasync.WorkPackage{}, fmt.Errorf("%w: workPackage: %v", aasync.ErrInvalid, err)
		}
		return wp, nil
	}
	id, err := stringField(params, "workPackageId")
	if err != nil {
		return aasync.WorkPackage{}, err
	}
	wp, ok := s.distributor.Package(id)
	if !ok {
		return aasync.WorkPackage{}, fmt.Errorf("%w: work package %s", aasync.ErrNotFound, id)
	}
	return wp, nil
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
		return "", fmt.Errorf("%w: %s is required", aasync.ErrInvalid, field)
	}
	return value, nil
}

func errResponse(id any, code any, message string) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &Error{Code: code, Message: message}}
}

func errResponseFrom(id any, err error) Response {
	switch {
	case errors.Is(err, aasync.ErrInvalid):
		return errResponse(id, CodeInvalidParams, err.Error())
	case errors.Is(err, aasync.ErrNotFound):
		return errResponse(id, CodeNotFound, err.Error())
	default:
		return errResponse(id, CodeInternalError, err.Error())
	}
}
