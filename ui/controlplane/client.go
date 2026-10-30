// Package controlplane is the console's typed client for the aa control plane.
//
// The console never imports module internals: it speaks the external HTTP/JSON
// control plane (contracts/openapi/control-plane-v1.yaml) through this client.
package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Status is the aggregate host status.
type Status struct {
	Projects          int `json:"projects"`
	ActiveAssignments int `json:"activeAssignments"`
	Nodes             int `json:"nodes"`
}

// Project is a project record summary.
type Project struct {
	ContractVersion string `json:"contractVersion,omitempty"`
	Kind            string `json:"kind,omitempty"`
	ID              string `json:"id"`
	Name            string `json:"name,omitempty"`
	State           string `json:"state,omitempty"`
	LayoutVersion   string `json:"layoutVersion,omitempty"`
}

// WorkPackage is a work-package record summary.
type WorkPackage struct {
	ContractVersion string   `json:"contractVersion,omitempty"`
	Kind            string   `json:"kind,omitempty"`
	ID              string   `json:"id"`
	Title           string   `json:"title,omitempty"`
	State           string   `json:"state,omitempty"`
	Acceptance      []string `json:"acceptance,omitempty"`
	DependsOn       []string `json:"dependsOn,omitempty"`
}

// TaskGraphNode is one node in the task graph read model.
type TaskGraphNode struct {
	WorkPackageID string   `json:"workPackageId"`
	DependsOn     []string `json:"dependsOn,omitempty"`
	Readiness     string   `json:"readiness"`
}

// TaskGraph is the task graph read model.
type TaskGraph struct {
	ContractVersion string          `json:"contractVersion,omitempty"`
	Kind            string          `json:"kind,omitempty"`
	ID              string          `json:"id"`
	ProjectID       string          `json:"projectId,omitempty"`
	Revision        int             `json:"revision,omitempty"`
	Nodes           []TaskGraphNode `json:"nodes"`
}

// Assignment is an assignment summary.
type Assignment struct {
	ID            string `json:"id"`
	WorkPackageID string `json:"workPackageId"`
	WorkerID      string `json:"workerId"`
	State         string `json:"state"`
	Attempt       int    `json:"attempt"`
}

// Node is a compute/sync node summary.
type Node struct {
	ID        string `json:"id"`
	Authority bool   `json:"authority"`
	State     string `json:"state"`
}

// Tool is a tool/plugin manifest summary.
type Tool struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Sandbox bool   `json:"sandbox,omitempty"`
}

// MemoryItem is an issue or strategy memory read-model entry.
type MemoryItem struct {
	ContractVersion string `json:"contractVersion,omitempty"`
	Kind            string `json:"kind,omitempty"`
	ID              string `json:"id"`
	ProjectID       string `json:"projectId,omitempty"`
	Title           string `json:"title,omitempty"`
	Status          string `json:"status,omitempty"`
	Severity        string `json:"severity,omitempty"`
	Lifecycle       string `json:"lifecycle,omitempty"`
	Guidance        string `json:"guidance,omitempty"`
}

// VisualizerSession is one browsable observation/visualizer session.
type VisualizerSession struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"projectId,omitempty"`
	Actors        []string `json:"actors,omitempty"`
	StartSequence int      `json:"startSequence"`
	EndSequence   int      `json:"endSequence"`
	EventCount    int      `json:"eventCount"`
	StartedAt     string   `json:"startedAt,omitempty"`
	EndedAt       string   `json:"endedAt,omitempty"`
}

// ObservationCount is one labelled count in an observation report.
type ObservationCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ObservationReport is the deterministic report over one session.
type ObservationReport struct {
	SessionID     string             `json:"sessionId"`
	Events        int                `json:"events"`
	FirstSequence int                `json:"firstSequence"`
	LastSequence  int                `json:"lastSequence"`
	BySourceType  []ObservationCount `json:"bySourceType"`
	ByConfidence  []ObservationCount `json:"byConfidence"`
	WorkPackages  []string           `json:"workPackages,omitempty"`
	Actors        []string           `json:"actors,omitempty"`
}

// Error is a governed control-plane error.
type Error struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message) }

// Client is the console's view of the control plane.
type Client interface {
	Status(ctx context.Context) (Status, error)
	Projects(ctx context.Context) ([]Project, error)
	Project(ctx context.Context, id string) (Project, error)
	CreateProject(ctx context.Context, id, name string) error
	WorkPackages(ctx context.Context, projectID string) ([]WorkPackage, error)
	Graph(ctx context.Context, projectID string) (TaskGraph, error)
	SubmitWork(ctx context.Context, projectID, title string, dependsOn []string) (string, error)
	Assignments(ctx context.Context) ([]Assignment, error)
	Approve(ctx context.Context, gateID string) error
	Reject(ctx context.Context, gateID, actor, reason string) error
	Decide(ctx context.Context, gateID, decision, actor, reason string) error
	Nodes(ctx context.Context) ([]Node, error)
	Tools(ctx context.Context) ([]Tool, error)
	Issues(ctx context.Context, projectID string) ([]MemoryItem, error)
	Strategies(ctx context.Context, projectID string) ([]MemoryItem, error)
	VisualizerSessions(ctx context.Context) ([]VisualizerSession, error)
	ObservationReport(ctx context.Context, sessionID string) (ObservationReport, error)
}

// HTTPClient is an Client over the control plane's HTTP/JSON API.
type HTTPClient struct {
	base  string
	token string
	http  *http.Client
}

// NewHTTPClient returns a client for baseURL (including the /api/v1 prefix).
func NewHTTPClient(baseURL, token string) *HTTPClient {
	return &HTTPClient{
		base:  strings.TrimRight(baseURL, "/"),
		token: token,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPClient) do(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := &Error{Status: resp.StatusCode}
		var envelope struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(data, &envelope) == nil {
			e.Code, e.Message = envelope.Error.Code, envelope.Error.Message
		}
		if e.Code == "" {
			e.Code, e.Message = "aa.error", strings.TrimSpace(string(data))
		}
		return e
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// Status implements Client.
func (c *HTTPClient) Status(ctx context.Context) (Status, error) {
	var s Status
	err := c.do(ctx, http.MethodGet, "/status", nil, &s)
	return s, err
}

// Projects implements Client.
func (c *HTTPClient) Projects(ctx context.Context) ([]Project, error) {
	var out []Project
	err := c.do(ctx, http.MethodGet, "/projects", nil, &out)
	return out, err
}

// Project implements Client.
func (c *HTTPClient) Project(ctx context.Context, id string) (Project, error) {
	var project Project
	err := c.do(ctx, http.MethodGet, "/projects/"+id, nil, &project)
	return project, err
}

// CreateProject implements Client.
func (c *HTTPClient) CreateProject(ctx context.Context, id, name string) error {
	return c.do(ctx, http.MethodPost, "/projects", map[string]string{"id": id, "name": name}, nil)
}

// WorkPackages implements Client.
func (c *HTTPClient) WorkPackages(ctx context.Context, projectID string) ([]WorkPackage, error) {
	var out []WorkPackage
	err := c.do(ctx, http.MethodGet, "/projects/"+projectID+"/work-packages", nil, &out)
	return out, err
}

// Graph implements Client.
func (c *HTTPClient) Graph(ctx context.Context, projectID string) (TaskGraph, error) {
	var graph TaskGraph
	err := c.do(ctx, http.MethodGet, "/projects/"+projectID+"/graph", nil, &graph)
	return graph, err
}

// SubmitWork implements Client.
func (c *HTTPClient) SubmitWork(ctx context.Context, projectID, title string, dependsOn []string) (string, error) {
	var out WorkPackage
	err := c.do(ctx, http.MethodPost, "/projects/"+projectID+"/work-packages", map[string]any{"title": title, "dependsOn": dependsOn}, &out)
	return out.ID, err
}

// Assignments implements Client.
func (c *HTTPClient) Assignments(ctx context.Context) ([]Assignment, error) {
	var out []Assignment
	err := c.do(ctx, http.MethodGet, "/assignments", nil, &out)
	return out, err
}

// Approve implements Client.
func (c *HTTPClient) Approve(ctx context.Context, gateID string) error {
	return c.Decide(ctx, gateID, "approve", "console", "")
}

// Reject records a rejection decision for a gate.
func (c *HTTPClient) Reject(ctx context.Context, gateID, actor, reason string) error {
	return c.Decide(ctx, gateID, "reject", actor, reason)
}

// Decide records an approval decision. Decision must be approve or reject.
func (c *HTTPClient) Decide(ctx context.Context, gateID, decision, actor, reason string) error {
	return c.do(ctx, http.MethodPost, "/approvals/"+gateID, map[string]string{
		"decision": decision,
		"actor":    actor,
		"reason":   reason,
	}, nil)
}

// Nodes implements Client.
func (c *HTTPClient) Nodes(ctx context.Context) ([]Node, error) {
	var out []Node
	err := c.do(ctx, http.MethodGet, "/nodes", nil, &out)
	return out, err
}

// Tools implements Client.
func (c *HTTPClient) Tools(ctx context.Context) ([]Tool, error) {
	var out []Tool
	err := c.do(ctx, http.MethodGet, "/tools", nil, &out)
	return out, err
}

// Issues implements Client.
func (c *HTTPClient) Issues(ctx context.Context, projectID string) ([]MemoryItem, error) {
	var out []MemoryItem
	err := c.do(ctx, http.MethodGet, "/projects/"+projectID+"/issues", nil, &out)
	return out, err
}

// Strategies implements Client.
func (c *HTTPClient) Strategies(ctx context.Context, projectID string) ([]MemoryItem, error) {
	var out []MemoryItem
	err := c.do(ctx, http.MethodGet, "/projects/"+projectID+"/strategies", nil, &out)
	return out, err
}

// VisualizerSessions implements Client.
func (c *HTTPClient) VisualizerSessions(ctx context.Context) ([]VisualizerSession, error) {
	var out []VisualizerSession
	err := c.do(ctx, http.MethodGet, "/visualizer/sessions", nil, &out)
	return out, err
}

// ObservationReport implements Client.
func (c *HTTPClient) ObservationReport(ctx context.Context, sessionID string) (ObservationReport, error) {
	var out ObservationReport
	err := c.do(ctx, http.MethodGet, "/observations/"+sessionID+"/report", nil, &out)
	return out, err
}
