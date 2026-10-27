package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/greadee/aa/memory/repo"
)

const testToken = "sprint2-token"

func startControlPlane(t *testing.T, root string) (*runtime, *httptest.Server) {
	t.Helper()
	g, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newControlPlane(g, testToken))
	return g, srv
}

func stopControlPlane(t *testing.T, g *runtime, srv *httptest.Server) {
	t.Helper()
	srv.Close()
	if err := g.obs.Close(); err != nil {
		t.Fatal(err)
	}
}

func apiRequest(t *testing.T, srv *httptest.Server, method, path, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testToken)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}

func decodeMap(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return out
}

func TestComposedControlPlaneRestartAndContractRegression(t *testing.T) {
	root := t.TempDir()
	g, srv := startControlPlane(t, root)

	status, body := apiRequest(t, srv, http.MethodPost, "/api/v1/projects", `{"id":"p1","name":"Project One"}`)
	if status != http.StatusCreated {
		t.Fatalf("create project = %d %s", status, body)
	}
	project := decodeMap(t, body)
	if project["kind"] != "project_record" || project["layoutVersion"] != "1" || project["state"] != "CREATED" {
		t.Fatalf("project shape = %+v", project)
	}
	if status, _ := apiRequest(t, srv, http.MethodPost, "/api/v1/projects", `{"id":"p1","name":"Duplicate"}`); status != http.StatusConflict {
		t.Fatalf("duplicate project = %d, want 409", status)
	}
	if status, _ := apiRequest(t, srv, http.MethodGet, "/api/v1/projects/missing", ""); status != http.StatusNotFound {
		t.Fatalf("missing project = %d, want 404", status)
	}

	status, body = apiRequest(t, srv, http.MethodPost, "/api/v1/projects/p1/work-packages", `{"title":"first"}`)
	if status != http.StatusCreated {
		t.Fatalf("submit first = %d %s", status, body)
	}
	first := decodeMap(t, body)
	firstID, _ := first["id"].(string)
	if firstID == "" || first["kind"] != "work_package" || first["state"] != "PROPOSED" {
		t.Fatalf("first work = %+v", first)
	}
	if _, ok := first["acceptance"].([]any); !ok {
		t.Fatalf("acceptance missing: %+v", first)
	}
	status, body = apiRequest(t, srv, http.MethodPost, "/api/v1/projects/p1/work-packages", `{"title":"second","dependsOn":["`+firstID+`"]}`)
	if status != http.StatusCreated {
		t.Fatalf("submit second = %d %s", status, body)
	}
	second := decodeMap(t, body)
	secondID, _ := second["id"].(string)
	if secondID == "" || secondID == firstID {
		t.Fatalf("second id = %q", secondID)
	}
	if status, _ := apiRequest(t, srv, http.MethodPost, "/api/v1/projects/missing/work-packages", `{"title":"x"}`); status != http.StatusBadRequest {
		t.Fatalf("unknown project submit = %d, want 400", status)
	}

	status, body = apiRequest(t, srv, http.MethodGet, "/api/v1/projects/p1/work-packages", "")
	if status != http.StatusOK {
		t.Fatalf("list work = %d %s", status, body)
	}
	var work []map[string]any
	if err := json.Unmarshal(body, &work); err != nil || len(work) != 2 {
		t.Fatalf("work = %s err=%v", body, err)
	}
	status, body = apiRequest(t, srv, http.MethodGet, "/api/v1/projects/p1/graph", "")
	if status != http.StatusOK {
		t.Fatalf("graph = %d %s", status, body)
	}
	graph := decodeMap(t, body)
	if graph["kind"] != "task_graph" || graph["projectId"] != "p1" {
		t.Fatalf("graph shape = %+v", graph)
	}
	nodes, _ := graph["nodes"].([]any)
	if len(nodes) != 2 {
		t.Fatalf("graph nodes = %+v", graph["nodes"])
	}
	if status, _ := apiRequest(t, srv, http.MethodGet, "/api/v1/projects/missing/graph", ""); status != http.StatusNotFound {
		t.Fatalf("missing graph = %d, want 404", status)
	}
	if status, _ := apiRequest(t, srv, http.MethodGet, "/api/v1/projects/p1/events", ""); status != http.StatusOK {
		t.Fatalf("events = %d, want 200", status)
	}

	assignments := repo.NewAssignmentRepository(g.store, g.projection)
	if _, err := assignments.Save(repo.AssignmentRecord{
		ID: "asg_p1_1", ProjectID: "p1", WorkPackageID: firstID,
		WorkerID: "w1", State: "awaiting_gates", Attempt: 1,
	}); err != nil {
		t.Fatal(err)
	}

	status, body = apiRequest(t, srv, http.MethodPost, "/api/v1/approvals/"+firstID, `{"decision":"approve","actor":"human","reason":"ok"}`)
	if status != http.StatusOK || !strings.Contains(string(body), `"approved"`) {
		t.Fatalf("approve = %d %s", status, body)
	}
	if status, _ := apiRequest(t, srv, http.MethodPost, "/api/v1/approvals/"+firstID, `{"decision":"approve","actor":"human"}`); status != http.StatusConflict {
		t.Fatalf("duplicate approval = %d, want 409", status)
	}
	status, body = apiRequest(t, srv, http.MethodPost, "/api/v1/approvals/"+secondID, `{"decision":"reject","actor":"human","reason":"no"}`)
	if status != http.StatusOK || !strings.Contains(string(body), `"rejected"`) {
		t.Fatalf("reject = %d %s", status, body)
	}
	if status, _ := apiRequest(t, srv, http.MethodPost, "/api/v1/approvals/"+firstID, ""); status != http.StatusBadRequest {
		t.Fatalf("empty approval body = %d, want 400", status)
	}
	if status, _ := apiRequest(t, srv, http.MethodPost, "/api/v1/approvals/missing", `{"decision":"approve","actor":"human"}`); status != http.StatusNotFound {
		t.Fatalf("missing gate = %d, want 404", status)
	}

	stopControlPlane(t, g, srv)

	g2, srv2 := startControlPlane(t, root)
	defer stopControlPlane(t, g2, srv2)

	if status, body := apiRequest(t, srv2, http.MethodGet, "/api/v1/projects/p1", ""); status != http.StatusOK || !strings.Contains(string(body), `"kind":"project_record"`) {
		t.Fatalf("project after restart = %d %s", status, body)
	}
	status, body = apiRequest(t, srv2, http.MethodGet, "/api/v1/projects/p1/work-packages", "")
	if status != http.StatusOK {
		t.Fatalf("work after restart = %d %s", status, body)
	}
	work = nil
	if err := json.Unmarshal(body, &work); err != nil || len(work) != 2 {
		t.Fatalf("work after restart = %s err=%v", body, err)
	}
	_, graphBody := apiRequest(t, srv2, http.MethodGet, "/api/v1/projects/p1/graph", "")
	graph = decodeMap(t, graphBody)
	if nodes, _ := graph["nodes"].([]any); len(nodes) != 2 {
		t.Fatalf("graph after restart = %+v", graph)
	}
	if statuses := g2.projects.AssignmentStatuses(); len(statuses) != 1 || statuses[0].State != "awaiting_gates" {
		t.Fatalf("assignments after restart = %+v", statuses)
	}
	if g2.projects.ActiveAssignments() != 1 {
		t.Fatalf("active assignments after restart = %d", g2.projects.ActiveAssignments())
	}
	approvals, err := g2.projects.ApprovalRecords()
	if err != nil || len(approvals) != 2 {
		t.Fatalf("approvals after restart = %+v err=%v", approvals, err)
	}
	if len(g2.projects.Projects()) != 2 {
		t.Fatalf("projects after restart = %+v", g2.projects.Projects())
	}
}
