package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClient(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/status":
			_ = json.NewEncoder(w).Encode(Status{Projects: 1, Nodes: 1})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects":
			_ = json.NewEncoder(w).Encode([]Project{{ContractVersion: "2.0", Kind: "project_record", ID: "default", Name: "Default", State: "CREATED", LayoutVersion: "1"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/default":
			_ = json.NewEncoder(w).Encode(Project{ContractVersion: "2.0", Kind: "project_record", ID: "default", Name: "Default", State: "CREATED", LayoutVersion: "1"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/default/graph":
			_ = json.NewEncoder(w).Encode(TaskGraph{ContractVersion: "2.0", Kind: "task_graph", ID: "graph_default", ProjectID: "default", Revision: 1, Nodes: []TaskGraphNode{{WorkPackageID: "wp-1", Readiness: "ready"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/default/work-packages":
			_ = json.NewEncoder(w).Encode([]WorkPackage{{ID: "wp-1", State: "PROPOSED"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/default/work-packages":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(WorkPackage{ContractVersion: "2.0", Kind: "work_package", ID: "wp-9", Title: "x", State: "PROPOSED", Acceptance: []string{"done"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/nodes":
			_ = json.NewEncoder(w).Encode([]Node{{ID: "local", Authority: true}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/tools":
			_ = json.NewEncoder(w).Encode([]Tool{{ID: "tool_screen"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/approvals/g1":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"aa.not_found","message":"nope"}}`))
		}
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL+"/api/v1", "tok")
	ctx := context.Background()

	if st, err := c.Status(ctx); err != nil || st.Nodes != 1 {
		t.Fatalf("status %+v %v", st, err)
	}
	if ps, err := c.Projects(ctx); err != nil || len(ps) != 1 || ps[0].ID != "default" {
		t.Fatalf("projects %v %v", ps, err)
	}
	if p, err := c.Project(ctx, "default"); err != nil || p.LayoutVersion != "1" {
		t.Fatalf("project %+v %v", p, err)
	}
	if g, err := c.Graph(ctx, "default"); err != nil || len(g.Nodes) != 1 || g.Nodes[0].Readiness != "ready" {
		t.Fatalf("graph %+v %v", g, err)
	}
	if err := c.CreateProject(ctx, "p", "n"); err != nil {
		t.Fatal(err)
	}
	if wps, err := c.WorkPackages(ctx, "default"); err != nil || len(wps) != 1 {
		t.Fatalf("work %v %v", wps, err)
	}
	if id, err := c.SubmitWork(ctx, "default", "x", nil); err != nil || id != "wp-9" {
		t.Fatalf("submit %q %v", id, err)
	}
	if err := c.Approve(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Nodes(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Tools(ctx); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("authorization = %q", gotAuth)
	}

	// Error decoding.
	var out any
	err := c.do(ctx, http.MethodGet, "/nope", nil, &out)
	var e *Error
	if !errors.As(err, &e) || e.Code != "aa.not_found" || e.Status != 404 {
		t.Fatalf("error = %v", err)
	}
}

func TestFakeClient(t *testing.T) {
	f := NewFake()
	ctx := context.Background()
	id, err := f.SubmitWork(ctx, "default", "first", nil)
	if err != nil || id == "" {
		t.Fatalf("submit %q %v", id, err)
	}
	wps, _ := f.WorkPackages(ctx, "default")
	if len(wps) != 1 || wps[0].State != "PROPOSED" {
		t.Fatalf("work %+v", wps)
	}
	if _, err := f.Project(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if g, err := f.Graph(ctx, "default"); err != nil || len(g.Nodes) != 1 {
		t.Fatalf("graph %+v %v", g, err)
	}
	st, _ := f.Status(ctx)
	if st.Projects == 0 || st.Nodes == 0 {
		t.Fatalf("status %+v", st)
	}
	if err := f.Approve(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
}
