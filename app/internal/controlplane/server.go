// Package controlplane serves the external, versioned HTTP/JSON control plane
// (contracts/openapi/control-plane-v1.yaml) from the composition root.
//
// It is a thin transport over the composed graph: handlers translate HTTP to the
// in-process services and back. It never holds execution authority beyond what
// the composed services grant, and it binds to loopback only.
package controlplane

import (
	"encoding/json"
	"net/http"
)

// APIPrefix is the base path from the OpenAPI `servers[0].url`.
const APIPrefix = "/api/v1"

// RPCVersion is the control-plane RPC generation advertised on /health.
const RPCVersion = "1.0"

// Status is the aggregate host status served on /status.
type Status struct {
	Projects          int `json:"projects"`
	ActiveAssignments int `json:"activeAssignments"`
	Nodes             int `json:"nodes"`
}

// Config configures the server.
type Config struct {
	// Version is the build version reported on /health.
	Version string
	// Token, when set, is required as `Authorization: Bearer <token>` on every
	// path except /health. Loopback alone is not authentication; this is only a
	// bearer-token carrying mechanism (real authn is a later phase).
	Token string
	// Status supplies the aggregate status. Optional.
	Status func() Status
	// Backend supplies projects/work/assignments/approvals. Optional.
	Backend Backend
}

// Server is the control-plane HTTP handler.
type Server struct {
	cfg Config
	mux *http.ServeMux
}

// New returns a control-plane handler.
func New(cfg Config) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	// /health is unauthenticated per the spec (security: []).
	s.mux.HandleFunc("GET "+APIPrefix+"/health", s.health)
	// Everything else carries the bearer token.
	s.mux.Handle("GET "+APIPrefix+"/status", s.auth(s.status))
	s.mux.Handle("GET "+APIPrefix+"/projects", s.auth(s.projectsList))
	s.mux.Handle("POST "+APIPrefix+"/projects", s.auth(s.projectsCreate))
	s.mux.Handle("GET "+APIPrefix+"/projects/{projectId}/work-packages", s.auth(s.workList))
	s.mux.Handle("POST "+APIPrefix+"/projects/{projectId}/work-packages", s.auth(s.workSubmit))
	s.mux.Handle("GET "+APIPrefix+"/projects/{projectId}/graph", s.auth(s.workList))
	s.mux.Handle("GET "+APIPrefix+"/assignments", s.auth(s.assignments))
	s.mux.Handle("POST "+APIPrefix+"/approvals/{gateId}", s.auth(s.approve))
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "aa.method_not_allowed", "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "aa.method_not_allowed", "method not allowed")
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
		writeError(w, http.StatusServiceUnavailable, "aa.unavailable", "no backend configured")
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "aa.invalid", "invalid JSON body")
		return
	}
	if err := s.cfg.Backend.CreateProject(req.ID, req.Name); err != nil {
		writeError(w, http.StatusBadRequest, "aa.invalid", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Project{ID: req.ID, Name: req.Name})
}

func (s *Server) workList(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []WorkPackageView{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.WorkPackages(r.PathValue("projectId")))
}

func (s *Server) workSubmit(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeError(w, http.StatusServiceUnavailable, "aa.unavailable", "no backend configured")
		return
	}
	var req struct {
		Title     string   `json:"title"`
		DependsOn []string `json:"dependsOn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "aa.invalid", "invalid JSON body")
		return
	}
	id, err := s.cfg.Backend.SubmitWork(r.PathValue("projectId"), req.Title, req.DependsOn)
	if err != nil {
		writeError(w, http.StatusBadRequest, "aa.invalid", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *Server) assignments(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Backend == nil {
		writeJSON(w, http.StatusOK, []AssignmentView{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Backend.Assignments())
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Backend == nil {
		writeError(w, http.StatusServiceUnavailable, "aa.unavailable", "no backend configured")
		return
	}
	gateID := r.PathValue("gateId")
	if err := s.cfg.Backend.Approve(gateID); err != nil {
		writeError(w, http.StatusBadRequest, "aa.invalid", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved", "gateId": gateID})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token != "" {
			if r.Header.Get("Authorization") != "Bearer "+s.cfg.Token {
				writeError(w, http.StatusUnauthorized, "aa.unauthorized", "missing or invalid bearer token")
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
