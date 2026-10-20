package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrDefinitionConflict is returned by a registry backend when a definition
// with the same id already exists.
var ErrDefinitionConflict = errors.New("controlplane: definition already exists")

// RegistryBackend serves the durable definition stores. It is implemented by
// the composition root; the transport never imports the registry module.
type RegistryBackend interface {
	Teams() []json.RawMessage
	Team(id string) (json.RawMessage, bool)
	AddTeam(spec json.RawMessage) error
	Routines() []json.RawMessage
	Routine(id string) (json.RawMessage, bool)
	AddRoutine(spec json.RawMessage) error
	Policies() []json.RawMessage
	Policy(id string) (json.RawMessage, bool)
	AddPolicy(spec json.RawMessage) error
}

func (s *Server) teamsList(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Registry == nil {
		writeJSON(w, http.StatusOK, []json.RawMessage{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Registry.Teams())
}

func (s *Server) teamGet(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Registry == nil {
		writeError(w, http.StatusNotFound, codeNotFound, "team not found")
		return
	}
	spec, ok := s.cfg.Registry.Team(r.PathValue("teamId"))
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "team not found")
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (s *Server) teamAdd(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Registry == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "registry is not configured")
		return
	}
	s.addDefinition(w, r, s.cfg.Registry.AddTeam)
}

func (s *Server) routinesList(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Registry == nil {
		writeJSON(w, http.StatusOK, []json.RawMessage{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Registry.Routines())
}

func (s *Server) routineGet(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Registry == nil {
		writeError(w, http.StatusNotFound, codeNotFound, "routine not found")
		return
	}
	spec, ok := s.cfg.Registry.Routine(r.PathValue("routineId"))
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "routine not found")
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (s *Server) routineAdd(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Registry == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "registry is not configured")
		return
	}
	s.addDefinition(w, r, s.cfg.Registry.AddRoutine)
}

func (s *Server) policiesList(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Registry == nil {
		writeJSON(w, http.StatusOK, []json.RawMessage{})
		return
	}
	writeJSON(w, http.StatusOK, s.cfg.Registry.Policies())
}

func (s *Server) policyGet(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Registry == nil {
		writeError(w, http.StatusNotFound, codeNotFound, "policy not found")
		return
	}
	spec, ok := s.cfg.Registry.Policy(r.PathValue("policyId"))
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "policy not found")
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (s *Server) policyAdd(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Registry == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "registry is not configured")
		return
	}
	s.addDefinition(w, r, s.cfg.Registry.AddPolicy)
}

func (s *Server) addDefinition(w http.ResponseWriter, r *http.Request, add func(json.RawMessage) error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "invalid definition body")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, codeInvalidParams, "definition body is required")
		return
	}
	if err := add(json.RawMessage(data)); err != nil {
		if errors.Is(err, ErrDefinitionConflict) {
			writeError(w, http.StatusConflict, codeConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}
