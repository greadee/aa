package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
)

// VisualizerSession is one browsable session summary.
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

// VisualizerNode is one node in a projected session graph.
type VisualizerNode struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Label    string            `json:"label,omitempty"`
	Ref      string            `json:"ref,omitempty"`
	Sequence int               `json:"sequence"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VisualizerEdge is one edge in a projected session graph.
type VisualizerEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Sequence int    `json:"sequence"`
}

// VisualizerGraph is the deterministic projection of a session.
type VisualizerGraph struct {
	SessionID string           `json:"sessionId"`
	Nodes     []VisualizerNode `json:"nodes"`
	Edges     []VisualizerEdge `json:"edges"`
}

// VisualizerFrame is the replay metadata of one frame.
type VisualizerFrame struct {
	Index    int    `json:"index"`
	Sequence int    `json:"sequence"`
	EventID  string `json:"eventId,omitempty"`
	Type     string `json:"type,omitempty"`
	Nodes    int    `json:"nodes"`
	Edges    int    `json:"edges"`
}

// VisualizerTimeline is a retained, deterministic replay frame list.
type VisualizerTimeline struct {
	SessionID string            `json:"sessionId"`
	Frames    []VisualizerFrame `json:"frames"`
}

// VisualizerBackend serves the composed visualizer read model. It is
// implemented by the composition root; the transport never imports the
// visualizer module.
type VisualizerBackend interface {
	Sessions(ctx context.Context) ([]VisualizerSession, error)
	Timeline(ctx context.Context, sessionID string) (VisualizerTimeline, error)
	Graph(ctx context.Context, sessionID string) (VisualizerGraph, error)
	Subscribe(ctx context.Context, sessionID string) (<-chan json.RawMessage, error)
}

func (s *Server) visualizerSessions(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Visualizer == nil {
		writeJSON(w, http.StatusOK, []VisualizerSession{})
		return
	}
	sessions, err := s.cfg.Visualizer.Sessions(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParams, err.Error())
		return
	}
	if sessions == nil {
		sessions = []VisualizerSession{}
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) visualizerTimeline(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Visualizer == nil {
		writeError(w, http.StatusNotFound, codeNotFound, "visualizer is not configured")
		return
	}
	timeline, err := s.cfg.Visualizer.Timeline(r.Context(), r.PathValue("sessionId"))
	if err != nil {
		writeError(w, http.StatusNotFound, codeNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}

func (s *Server) visualizerGraph(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Visualizer == nil {
		writeError(w, http.StatusNotFound, codeNotFound, "visualizer is not configured")
		return
	}
	graph, err := s.cfg.Visualizer.Graph(r.Context(), r.PathValue("sessionId"))
	if err != nil {
		writeError(w, http.StatusNotFound, codeNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) visualizerStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, codeUnavailable, "streaming is unavailable")
		return
	}
	if s.cfg.Visualizer == nil {
		writeError(w, http.StatusServiceUnavailable, codeUnavailable, "visualizer is not configured")
		return
	}
	events, err := s.cfg.Visualizer.Subscribe(r.Context(), r.PathValue("sessionId"))
	if err != nil {
		writeError(w, http.StatusNotFound, codeNotFound, err.Error())
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
			if _, err := w.Write([]byte("event: visualizer\ndata: " + string(ev) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
