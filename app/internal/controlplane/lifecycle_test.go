package controlplane

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeLifecycle struct{}

func (fakeLifecycle) Workers() []WorkerLifecycle {
	return []WorkerLifecycle{{ID: "wrk_asg_1", SelectedWorker: "w1", AssignmentID: "asg_1", State: "stopped"}}
}

func (fakeLifecycle) WorkerEvents() []WorkerLifecycleEvent {
	return []WorkerLifecycleEvent{{WorkerID: "wrk_asg_1", State: "stopped", Reason: "completed"}}
}

func TestLifecycleEndpoints(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Lifecycle: fakeLifecycle{}})
	for _, path := range []string{"/workers", "/workers/events"} {
		req := httptest.NewRequest(http.MethodGet, APIPrefix+path, nil)
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s = %d (%s)", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "wrk_asg_1") {
			t.Fatalf("%s body = %s", path, rec.Body.String())
		}
	}
}

func TestLifecycleUnconfigured(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/workers", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}
}
