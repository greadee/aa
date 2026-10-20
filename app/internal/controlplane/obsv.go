package controlplane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// Observation is one sanitized observation event served from the obsv journal.
type Observation struct {
	Version       string            `json:"version"`
	SessionID     string            `json:"sessionId"`
	Sequence      int               `json:"sequence"`
	OccurredAt    string            `json:"occurredAt,omitempty"`
	SourceType    string            `json:"sourceType"`
	Source        string            `json:"source,omitempty"`
	Action        string            `json:"action,omitempty"`
	Confidence    string            `json:"sourceConfidence"`
	Actor         string            `json:"actor,omitempty"`
	WorkPackageID string            `json:"workPackageId,omitempty"`
	AttemptID     string            `json:"attemptId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ObservationCount is a labelled count in a work report.
type ObservationCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ObservationReport is a deterministic report over one observation session.
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

// CodexIngest reports how many translated events were journaled or dropped.
type CodexIngest struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"`
}

// ObservationBackend serves the composed obsv journal. It is implemented by the
// composition root; the transport never imports the obsv module.
type ObservationBackend interface {
	Replay(sessionID string, from int) ([]Observation, error)
	Report(sessionID string) (ObservationReport, error)
	Subscribe(ctx context.Context, sessionID string, from int) (<-chan Observation, error)
	IngestCodex(sessionID string, r io.Reader) (CodexIngest, error)
}

func (s *Server) observations(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Observations == nil {
		writeJSON(w, http.StatusOK, []Observation{})
		return
	}
	from := 0
	if raw := r.URL.Query().Get("after"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, codeInvalidParams, "after must be a non-negative integer")
			return
		}
		from = n
	}
	events, err := s.cfg.Observations.Replay(r.PathValue("sessionId"), from)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		return
	}
	if events == nil {
		events = []Observation{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) observationReport(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Observations == nil {
		writeJSON(w, http.StatusOK, ObservationReport{})
		return
	}
	report, err := s.cfg.Observations.Report(r.PathValue("sessionId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) observationStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, codeUnavailable, "streaming is unavailable")
		return
	}
	if s.cfg.Observations == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "observations are not configured")
		return
	}
	from := 0
	if raw := r.URL.Query().Get("after"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, codeInvalidParams, "after must be a non-negative integer")
			return
		}
		from = n
	}
	events, err := s.cfg.Observations.Subscribe(r.Context(), r.PathValue("sessionId"), from)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := w.Write([]byte("event: observation\ndata: " + string(data) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) observationCodex(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Observations == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "observations are not configured")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	defer r.Body.Close()
	ingested, err := s.cfg.Observations.IngestCodex(r.PathValue("sessionId"), r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ingested)
}
