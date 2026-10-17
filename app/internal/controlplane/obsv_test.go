package controlplane

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeObservations struct{}

func (fakeObservations) Replay(sessionID string, from int) ([]Observation, error) {
	return []Observation{{
		Version:    "1.0",
		SessionID:  sessionID,
		Sequence:   from,
		SourceType: "tool",
		Confidence: "exact",
		Action:     "tool.run",
	}}, nil
}

func (fakeObservations) Report(sessionID string) (ObservationReport, error) {
	return ObservationReport{SessionID: sessionID, Events: 1, LastSequence: 0}, nil
}

func (fakeObservations) Subscribe(_ context.Context, sessionID string, from int) (<-chan Observation, error) {
	out := make(chan Observation, 1)
	out <- Observation{Version: "1.0", SessionID: sessionID, Sequence: from, SourceType: "tool", Confidence: "exact"}
	close(out)
	return out, nil
}

func (fakeObservations) IngestCodex(string, io.Reader) (CodexIngest, error) {
	return CodexIngest{Accepted: 1}, nil
}

func TestObservationEndpoints(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Observations: fakeObservations{}})

	cases := []struct {
		method, path, body string
		want               int
		contains           string
	}{
		{http.MethodGet, APIPrefix + "/observations/s1", "", http.StatusOK, `"tool.run"`},
		{http.MethodGet, APIPrefix + "/observations/s1?after=3", "", http.StatusOK, `"sequence":3`},
		{http.MethodGet, APIPrefix + "/observations/s1?after=nope", "", http.StatusBadRequest, "non-negative"},
		{http.MethodGet, APIPrefix + "/observations/s1/report", "", http.StatusOK, `"events":1`},
		{http.MethodPost, APIPrefix + "/observations/s1/codex", `{"type":"tool_call"}`, http.StatusOK, `"accepted":1`},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d (%s)", c.method, c.path, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.contains != "" && !strings.Contains(rec.Body.String(), c.contains) {
			t.Errorf("%s %s body = %s, want %q", c.method, c.path, rec.Body.String(), c.contains)
		}
	}
}

func TestObservationStream(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Observations: fakeObservations{}})
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/observations/s1/stream", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "event: observation") || !strings.Contains(rec.Body.String(), `"sessionId":"s1"`) {
		t.Fatalf("stream body = %q", rec.Body.String())
	}
}

func TestObservationEndpointsRequireToken(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Observations: fakeObservations{}})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/observations/s1", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestObservationUnconfiguredIsEmpty(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/observations/s1", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}
}
