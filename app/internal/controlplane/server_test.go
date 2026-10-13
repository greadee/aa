package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthUnauthenticated(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Status: func() Status { return Status{Nodes: 1} }})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["version"] != "0.1.0" || body["rpcVersion"] != RPCVersion {
		t.Fatalf("body = %v", body)
	}
}

func TestStatusRequiresToken(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Status: func() Status { return Status{Projects: 2, Nodes: 1} }})

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/status", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/status", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var st Status
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st.Projects != 2 || st.Nodes != 1 {
		t.Fatalf("status = %+v", st)
	}
}
