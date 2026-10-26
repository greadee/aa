package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

type fakeBackend struct{}

func (fakeBackend) Projects() []Project { return []Project{{ID: "p1", Name: "Alpha"}} }
func (fakeBackend) CreateProject(id, name string) error {
	if id == "" {
		return errors.New("bad id")
	}
	return nil
}
func (fakeBackend) WorkPackages(projectID string) []WorkPackageView {
	if projectID != "p1" {
		return nil
	}
	return []WorkPackageView{{ID: "wp-1", Title: "first", State: "PROPOSED"}}
}
func (fakeBackend) SubmitWork(projectID, title string, dependsOn []string) (string, error) {
	if projectID != "p1" {
		return "", errors.New("unknown project")
	}
	return "wp-1", nil
}
func (fakeBackend) Assignments() []AssignmentView                        { return []AssignmentView{} }
func (fakeBackend) Approve(gateID, decision, actor, reason string) error { return nil }
func (fakeBackend) Issues(string) []json.RawMessage                      { return nil }
func (fakeBackend) Strategies(string) []json.RawMessage                  { return nil }
func (fakeBackend) Events(string) []json.RawMessage                      { return nil }
func (fakeBackend) Nodes() []Node {
	return []Node{{ID: "local", Authority: true, State: "ready"}}
}
func (fakeBackend) SyncStatus() SyncStatus {
	return SyncStatus{
		LocalNode: "local",
		Nodes:     []SyncNode{{ID: "local", Local: true, State: "ready"}},
		Transfers: []SyncTransfer{},
	}
}
func (fakeBackend) Tools() []Tool {
	return []Tool{{ID: "tool_screen", Kind: "builtin"}}
}

func withToken(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer secret")
	return req
}

func TestResourcesRequireTokenAndServe(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Backend: fakeBackend{}})

	// Missing token is rejected.
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/projects", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("projects without token = %d", rec.Code)
	}

	// Projects list.
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/projects", nil)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Alpha") {
		t.Fatalf("projects = %d %s", rec.Code, rec.Body.String())
	}

	// Work-package submit + list.
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodPost, APIPrefix+"/projects/p1/work-packages", strings.NewReader(`{"title":"first"}`))))
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), "wp-1") {
		t.Fatalf("submit = %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/projects/p1/work-packages", nil)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "PROPOSED") {
		t.Fatalf("work list = %d %s", rec.Code, rec.Body.String())
	}

	// Nodes + tools.
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/nodes", nil)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "local") {
		t.Fatalf("nodes = %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/sync/status", nil)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "localNode") {
		t.Fatalf("sync status = %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/tools", nil)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "tool_screen") {
		t.Fatalf("tools = %d %s", rec.Code, rec.Body.String())
	}
}
