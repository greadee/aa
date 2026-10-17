package controlplane

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenAPIPaths exercises every path in contracts/openapi/control-plane-v1.yaml
// against the handler with a deterministic backend.
func TestOpenAPIPaths(t *testing.T) {
	s := New(Config{
		Version: "0.1.0",
		Token:   "t",
		Status:  func() Status { return Status{Nodes: 1} },
		Backend: fakeBackend{},
	})
	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, APIPrefix + "/health", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/status", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/projects", "", http.StatusOK},
		{http.MethodPost, APIPrefix + "/projects", `{"id":"p1","name":"Alpha"}`, http.StatusCreated},
		{http.MethodGet, APIPrefix + "/projects/p1/work-packages", "", http.StatusOK},
		{http.MethodPost, APIPrefix + "/projects/p1/work-packages", `{"title":"x"}`, http.StatusCreated},
		{http.MethodGet, APIPrefix + "/projects/p1/graph", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/projects/p1/issues", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/projects/p1/strategies", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/projects/p1/events", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/assignments", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/nodes", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/sync/status", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/tools", "", http.StatusOK},
		{http.MethodPost, APIPrefix + "/approvals/g1", "", http.StatusOK},
		{http.MethodGet, APIPrefix + "/nope", "", http.StatusNotFound},
	}
	for _, c := range cases {
		var body io.Reader
		if c.body != "" {
			body = strings.NewReader(c.body)
		}
		req := httptest.NewRequest(c.method, c.path, body)
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d (%s)", c.method, c.path, rec.Code, c.want, strings.TrimSpace(rec.Body.String()))
		}
	}
}
