package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeVisualizer struct{}

func (fakeVisualizer) Sessions(context.Context) ([]VisualizerSession, error) {
	return []VisualizerSession{{ID: "s1", EventCount: 2}}, nil
}

func (fakeVisualizer) Timeline(_ context.Context, sessionID string) (VisualizerTimeline, error) {
	return VisualizerTimeline{
		SessionID: sessionID,
		Frames:    []VisualizerFrame{{Index: 0, Sequence: 0, EventID: "e1", Nodes: 2, Edges: 1}},
	}, nil
}

func (fakeVisualizer) Graph(_ context.Context, sessionID string) (VisualizerGraph, error) {
	return VisualizerGraph{
		SessionID: sessionID,
		Nodes:     []VisualizerNode{{ID: "session:s1", Kind: "session"}},
		Edges:     []VisualizerEdge{},
	}, nil
}

func (fakeVisualizer) Subscribe(_ context.Context, sessionID string) (<-chan json.RawMessage, error) {
	out := make(chan json.RawMessage, 1)
	out <- json.RawMessage(`{"sessionId":"` + sessionID + `"}`)
	close(out)
	return out, nil
}

func TestVisualizerEndpoints(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Visualizer: fakeVisualizer{}})

	cases := []struct {
		method, path string
		want         int
		contains     string
	}{
		{http.MethodGet, APIPrefix + "/visualizer/sessions", http.StatusOK, `"eventCount":2`},
		{http.MethodGet, APIPrefix + "/visualizer/sessions/s1/timeline", http.StatusOK, `"nodes":2`},
		{http.MethodGet, APIPrefix + "/visualizer/sessions/s1/graph", http.StatusOK, `"session:s1"`},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d (%s)", c.method, c.path, rec.Code, c.want, rec.Body.String())
			continue
		}
		if !strings.Contains(rec.Body.String(), c.contains) {
			t.Errorf("%s %s body = %s, want %q", c.method, c.path, rec.Body.String(), c.contains)
		}
	}
}

func TestVisualizerStream(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Visualizer: fakeVisualizer{}})
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/visualizer/sessions/s1/stream", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "event: visualizer") || !strings.Contains(rec.Body.String(), `"sessionId":"s1"`) {
		t.Fatalf("stream body = %q", rec.Body.String())
	}
}

func TestVisualizerUnconfigured(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/visualizer/sessions", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}
}
