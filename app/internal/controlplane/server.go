// Package controlplane serves the external, versioned HTTP/JSON control plane
// (contracts/openapi/control-plane-v1.yaml) from the composition root.
//
// It is a thin transport over the composed graph: handlers translate HTTP to the
// in-process services and back. It never holds execution authority beyond what
// the composed services grant, and it binds to loopback only.
package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

// APIPrefix is the base path from the OpenAPI `servers[0].url`.
const APIPrefix = "/api/v1"

// RPCVersion is the control-plane RPC generation advertised on /health.
const RPCVersion = "1.0"

// Backend error classes mapped to governed HTTP errors.
var (
	ErrInvalid  = errors.New("controlplane: invalid request")
	ErrNotFound = errors.New("controlplane: not found")
	ErrConflict = errors.New("controlplane: conflict")
)

// Status is the aggregate host status served on /status.
type Status struct {
	Projects          int `json:"projects"`
	ActiveAssignments int `json:"activeAssignments"`
	Nodes             int `json:"nodes"`
}

// Config configures the server.
type Config struct {
	// Version is the build version reported on /api/v1/health.
	Version string
	// Token, when set, is required as `Authorization: Bearer <token>` on every
	// path except /api/v1/health. An empty token disables the check. Loopback
	// alone is not authentication; this is only a
	// bearer-token carrying mechanism (real authn is a later phase).
	Token string
	// Status supplies the aggregate status. Optional.
	Status func() Status
	// Backend supplies projects/work/assignments/approvals. Optional.
	Backend Backend
	// Observations supplies the composed obsv journal. Optional.
	Observations ObservationBackend
	// Visualizer supplies the composed visualizer read model. Optional.
	Visualizer VisualizerBackend
	// Registry supplies the durable definition stores. Optional.
	Registry RegistryBackend
	// Lifecycle supplies the runtime worker lifecycle read model. Optional.
	Lifecycle LifecycleBackend
}

// Server is the control-plane HTTP handler.
type Server struct {
	cfg    Config
	mux    *http.ServeMux
	routes []string
}

// New returns a control-plane handler.
func New(cfg Config) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	routes := []struct {
		pattern string
		handler http.HandlerFunc
		public  bool
	}{
		// /health is unauthenticated per the spec (security: []).
		{"GET " + APIPrefix + "/health", s.health, true},
		{"GET " + APIPrefix + "/status", s.status, false},
		{"GET " + APIPrefix + "/projects", s.projectsList, false},
		{"POST " + APIPrefix + "/projects", s.projectsCreate, false},
		{"GET " + APIPrefix + "/projects/{projectId}", s.projectGet, false},
		{"GET " + APIPrefix + "/projects/{projectId}/work-packages", s.workList, false},
		{"POST " + APIPrefix + "/projects/{projectId}/work-packages", s.workSubmit, false},
		{"GET " + APIPrefix + "/projects/{projectId}/graph", s.graph, false},
		{"GET " + APIPrefix + "/projects/{projectId}/issues", s.issues, false},
		{"GET " + APIPrefix + "/projects/{projectId}/strategies", s.strategies, false},
		{"GET " + APIPrefix + "/projects/{projectId}/events", s.events, false},
		{"GET " + APIPrefix + "/assignments", s.assignments, false},
		{"GET " + APIPrefix + "/nodes", s.nodes, false},
		{"GET " + APIPrefix + "/sync/status", s.syncStatus, false},
		{"GET " + APIPrefix + "/observations/{sessionId}", s.observations, false},
		{"GET " + APIPrefix + "/observations/{sessionId}/stream", s.observationStream, false},
		{"GET " + APIPrefix + "/observations/{sessionId}/report", s.observationReport, false},
		{"POST " + APIPrefix + "/observations/{sessionId}/codex", s.observationCodex, false},
		{"GET " + APIPrefix + "/visualizer/sessions", s.visualizerSessions, false},
		{"GET " + APIPrefix + "/visualizer/sessions/{sessionId}/timeline", s.visualizerTimeline, false},
		{"GET " + APIPrefix + "/visualizer/sessions/{sessionId}/graph", s.visualizerGraph, false},
		{"GET " + APIPrefix + "/visualizer/sessions/{sessionId}/stream", s.visualizerStream, false},
		{"GET " + APIPrefix + "/teams", s.teamsList, false},
		{"POST " + APIPrefix + "/teams", s.teamAdd, false},
		{"GET " + APIPrefix + "/teams/{teamId}", s.teamGet, false},
		{"GET " + APIPrefix + "/routines", s.routinesList, false},
		{"POST " + APIPrefix + "/routines", s.routineAdd, false},
		{"GET " + APIPrefix + "/routines/{routineId}", s.routineGet, false},
		{"GET " + APIPrefix + "/policies", s.policiesList, false},
		{"POST " + APIPrefix + "/policies", s.policyAdd, false},
		{"GET " + APIPrefix + "/policies/{policyId}", s.policyGet, false},
		{"GET " + APIPrefix + "/workers", s.workers, false},
		{"GET " + APIPrefix + "/workers/events", s.workerEvents, false},
		{"GET " + APIPrefix + "/tools", s.tools, false},
		{"POST " + APIPrefix + "/approvals/{gateId}", s.approve, false},
	}
	for _, route := range routes {
		s.routes = append(s.routes, route.pattern)
		if route.public {
			s.mux.HandleFunc(route.pattern, route.handler)
			continue
		}
		s.mux.Handle(route.pattern, s.auth(route.handler))
	}
	// Anything else under the prefix is not found (JSON, same error vocabulary).
	s.mux.HandleFunc(APIPrefix+"/", s.notFound)
	return s
}

// Routes returns the registered route patterns in stable order.
func (s *Server) Routes() []string {
	return append([]string(nil), s.routes...)
}

func (s *Server) notFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, codeNotFound, "not found")
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, codeMethodNotFound, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "ok",
		"version":    s.cfg.Version,
		"rpcVersion": RPCVersion,
	})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, codeMethodNotFound, "method not allowed")
		return
	}
	st := Status{}
	if s.cfg.Status != nil {
		st = s.cfg.Status()
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) projectsList(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []Project{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Projects())
}

func (s *Server) projectsCreate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "no backend configured")
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "invalid JSON body")
		return
	}
	if err := s.cfg.Backend.CreateProject(req.ID, req.Name); err != nil {
		switch {
		case errors.Is(err, ErrConflict):
			writeError(w, http.StatusConflict, codeConflict, err.Error())
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, codeNotFound, err.Error())
		case errors.Is(err, ErrInvalid):
			writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		default:
			writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		}
		return
	}
	project, ok := s.cfg.Backend.Project(req.ID)
	if !ok {
		writeError(w, http.StatusInternalServerError, codeUnavailable, "project was created but could not be read")
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) projectGet(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeError(w, http.StatusNotFound, codeNotFound, "project not found")
		return
	}
	project, ok := s.cfg.Backend.Project(r.PathValue("projectId"))
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) workList(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []WorkPackageView{})
		return
	}
	projectID := r.PathValue("projectId")
	if _, ok := s.cfg.Backend.Project(projectID); !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "project not found")
		return
	}
	work := s.cfg.Backend.WorkPackages(projectID)
	if work == nil {
		work = []WorkPackageView{}
	}
	writeJSON(w, http.StatusOK, work)
}

func (s *Server) workSubmit(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "no backend configured")
		return
	}
	var req struct {
		Title     string   `json:"title"`
		DependsOn []string `json:"dependsOn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "invalid JSON body")
		return
	}
	projectID := r.PathValue("projectId")
	if _, ok := s.cfg.Backend.Project(projectID); !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "project not found")
		return
	}
	id, err := s.cfg.Backend.SubmitWork(projectID, req.Title, req.DependsOn)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, codeNotFound, err.Error())
		case errors.Is(err, ErrConflict):
			writeError(w, http.StatusConflict, codeConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		}
		return
	}
	work, ok := s.cfg.Backend.WorkPackage(projectID, id)
	if !ok {
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
		return
	}
	writeJSON(w, http.StatusCreated, work)
}

func (s *Server) graph(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, v2.TaskGraph{})
		return
	}
	graph, ok := s.cfg.Backend.Graph(r.PathValue("projectId"))
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) assignments(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []AssignmentView{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Assignments())
}

func (s *Server) issues(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []json.RawMessage{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Issues(r.PathValue("projectId")))
}

func (s *Server) strategies(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []json.RawMessage{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Strategies(r.PathValue("projectId")))
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []json.RawMessage{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Events(r.PathValue("projectId")))
}

func (s *Server) nodes(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []Node{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Nodes())
}

func (s *Server) syncStatus(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, SyncStatus{Nodes: []SyncNode{}, Transfers: []SyncTransfer{}})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.SyncStatus())
}

func (s *Server) tools(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []Tool{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Tools())
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "no backend configured")
		return
	}
	var req struct {
		Decision string `json:"decision"`
		Actor    string `json:"actor"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "invalid JSON body")
		return
	}
	if req.Decision != "approve" && req.Decision != "reject" {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "decision must be approve or reject")
		return
	}
	if req.Actor == "" {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "actor is required")
		return
	}
	gateID := r.PathValue("gateId")
	if err := s.cfg.Backend.Approve(gateID, req.Decision, req.Actor, req.Reason); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, codeNotFound, err.Error())
		case errors.Is(err, ErrConflict):
			writeError(w, http.StatusConflict, codeConflict, err.Error())
		case errors.Is(err, ErrInvalid):
			writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		default:
			writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		}
		return
	}
	status := "approved"
	if req.Decision == "reject" {
		status = "rejected"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status, "gateId": gateID})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token != "" {
			if r.Header.Get("Authorization") != "Bearer "+s.cfg.Token {
				writeError(w, http.StatusUnauthorized, codeUnauthorized, "missing or invalid bearer token")
				return
			}
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
