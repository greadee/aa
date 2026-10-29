package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
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

func (fakeBackend) Projects() []Project {
	return []Project{{ContractVersion: "2.0", Kind: "project_record", ID: "p1", Name: "Alpha", State: "CREATED", LayoutVersion: "1"}}
}
func (fakeBackend) Project(id string) (Project, bool) {
	switch id {
	case "p1":
		return Project{ContractVersion: "2.0", Kind: "project_record", ID: "p1", Name: "Alpha", State: "CREATED", LayoutVersion: "1"}, true
	case "empty":
		return Project{ContractVersion: "2.0", Kind: "project_record", ID: "empty", Name: "Empty", State: "CREATED", LayoutVersion: "1"}, true
	default:
		return Project{}, false
	}
}
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
	return []WorkPackageView{{ContractVersion: "2.0", Kind: "work_package", ID: "wp-1", Title: "first", State: "PROPOSED", Acceptance: []string{"done"}}}
}

func TestWorkListUnknownAndEmptyProject(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Backend: fakeBackend{}})

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/projects/nope/work-packages", nil)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown project = %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"aa.not_found"`) {
		t.Fatalf("unknown project body = %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodGet, APIPrefix+"/projects/empty/work-packages", nil)))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty project = %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, withToken(httptest.NewRequest(http.MethodPost, APIPrefix+"/projects/nope/work-packages", strings.NewReader(`{"title":"x"}`))))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("submit unknown project = %d %s", rec.Code, rec.Body.String())
	}
}
func (fakeBackend) WorkPackage(projectID, id string) (WorkPackageView, bool) {
	if projectID != "p1" || id != "wp-1" {
		return WorkPackageView{}, false
	}
	return WorkPackageView{ContractVersion: "2.0", Kind: "work_package", ID: "wp-1", Title: "first", State: "PROPOSED", Acceptance: []string{"done"}}, true
}
func (fakeBackend) Graph(projectID string) (v2.TaskGraph, bool) {
	if projectID != "p1" {
		return v2.TaskGraph{}, false
	}
	rev := 1
	return v2.TaskGraph{
		Envelope: v2.Envelope{ContractVersion: "2.0", Kind: "task_graph", ID: "graph_p1", ProjectID: "p1", Revision: &rev},
		Nodes:    []v2.TaskGraphNode{{WorkPackageID: "wp-1", Readiness: v2.ReadinessReady}},
	}, true
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

type createBackend struct {
	fakeBackend
	err error
}

func (b createBackend) CreateProject(string, string) error { return b.err }

func TestProjectCreateErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{ErrConflict, http.StatusConflict},
		{ErrInvalid, http.StatusBadRequest},
	}
	for _, c := range cases {
		s := New(Config{Version: "0.1.0", Token: "secret", Status: func() Status { return Status{} }, Backend: createBackend{err: c.err}})
		req := httptest.NewRequest(http.MethodPost, APIPrefix+"/projects", strings.NewReader(`{"id":"p1","name":"Alpha"}`))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.code {
			t.Fatalf("err %v = %d %s", c.err, rec.Code, rec.Body.String())
		}
	}
}
