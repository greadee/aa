package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if err := f.Approve(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func TestClientSemanticsParity(t *testing.T) {
	srv := newSemanticServer()
	defer srv.Close()

	run := func(t *testing.T, c Client) {
		t.Helper()
		ctx := context.Background()
		if err := c.CreateProject(ctx, "p1", "One"); err != nil {
			t.Fatalf("create project: %v", err)
		}
		var e *Error
		if err := c.CreateProject(ctx, "p1", "Duplicate"); !errors.As(err, &e) || e.Code != "aa.conflict" {
			t.Fatalf("duplicate project err = %v", err)
		}
		if _, err := c.Project(ctx, "missing"); !errors.As(err, &e) || e.Code != "aa.not_found" {
			t.Fatalf("unknown project err = %v", err)
		}
		if _, err := c.WorkPackages(ctx, "missing"); !errors.As(err, &e) || e.Code != "aa.not_found" {
			t.Fatalf("unknown work list err = %v", err)
		}
		if _, err := c.SubmitWork(ctx, "p1", "", nil); !errors.As(err, &e) || e.Code != "aa.invalid_params" {
			t.Fatalf("empty title err = %v", err)
		}
		if _, err := c.SubmitWork(ctx, "p1", "work", []string{"missing"}); !errors.As(err, &e) || e.Code != "aa.invalid_params" {
			t.Fatalf("unknown dependency err = %v", err)
		}
		id, err := c.SubmitWork(ctx, "p1", "work", nil)
		if err != nil || id == "" {
			t.Fatalf("submit work = %q %v", id, err)
		}
		if err := c.Reject(ctx, id, "human", "no"); err != nil {
			t.Fatalf("reject: %v", err)
		}
		if err := c.Decide(ctx, id, "approve", "human", ""); !errors.As(err, &e) || e.Code != "aa.conflict" {
			t.Fatalf("duplicate decision err = %v", err)
		}
		if err := c.Decide(ctx, "missing", "approve", "human", ""); !errors.As(err, &e) || e.Code != "aa.not_found" {
			t.Fatalf("unknown gate err = %v", err)
		}
	}

	t.Run("fake", func(t *testing.T) { run(t, NewFake()) })
	t.Run("http", func(t *testing.T) { run(t, NewHTTPClient(srv.URL+"/api/v1", "tok")) })
}

func TestHTTPClientRejectPayload(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/approvals/g1" {
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL+"/api/v1", "tok")
	if err := c.Reject(context.Background(), "g1", "human", "not good"); err != nil {
		t.Fatal(err)
	}
	if got["decision"] != "reject" || got["actor"] != "human" || got["reason"] != "not good" {
		t.Fatalf("payload = %+v", got)
	}
}

func newSemanticServer() *httptest.Server {
	projects := map[string]bool{"default": true}
	work := map[string]WorkPackage{}
	decisions := map[string]bool{}
	seq := 0
	writeError := func(w http.ResponseWriter, status int, code, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodPost && path == "/api/v1/projects":
			var req struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.ID == "" {
				writeError(w, 400, "aa.invalid_params", "id required")
				return
			}
			if projects[req.ID] {
				writeError(w, 409, "aa.conflict", "exists")
				return
			}
			projects[req.ID] = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/projects/") && !strings.Contains(strings.TrimPrefix(path, "/api/v1/projects/"), "/"):
			id := strings.TrimPrefix(path, "/api/v1/projects/")
			if !projects[id] {
				writeError(w, 404, "aa.not_found", "project not found")
				return
			}
			_ = json.NewEncoder(w).Encode(Project{ID: id})
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/work-packages"):
			id := strings.Split(path, "/")[4]
			if !projects[id] {
				writeError(w, 404, "aa.not_found", "project not found")
				return
			}
			_ = json.NewEncoder(w).Encode([]WorkPackage{})
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/work-packages"):
			id := strings.Split(path, "/")[4]
			if !projects[id] {
				writeError(w, 404, "aa.not_found", "project not found")
				return
			}
			var req struct {
				Title     string   `json:"title"`
				DependsOn []string `json:"dependsOn"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Title == "" {
				writeError(w, 400, "aa.invalid_params", "title required")
				return
			}
			for _, dep := range req.DependsOn {
				if _, ok := work[dep]; !ok {
					writeError(w, 400, "aa.invalid_params", "unknown dependency")
					return
				}
			}
			seq++
			wp := WorkPackage{ContractVersion: "2.0", Kind: "work_package", ID: fmt.Sprintf("wp-%d", seq), Title: req.Title, State: "PROPOSED"}
			work[wp.ID] = wp
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(wp)
		case r.Method == http.MethodPost && strings.HasPrefix(path, "/api/v1/approvals/"):
			gate := strings.TrimPrefix(path, "/api/v1/approvals/")
			var req struct {
				Decision string `json:"decision"`
				Actor    string `json:"actor"`
				Reason   string `json:"reason"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Decision != "approve" && req.Decision != "reject" {
				writeError(w, 400, "aa.invalid_params", "decision")
				return
			}
			if req.Actor == "" {
				writeError(w, 400, "aa.invalid_params", "actor")
				return
			}
			if _, ok := work[gate]; !ok {
				writeError(w, 404, "aa.not_found", "work package not found")
				return
			}
			if decisions[gate] {
				writeError(w, 409, "aa.conflict", "already decided")
				return
			}
			decisions[gate] = true
			w.WriteHeader(http.StatusOK)
		default:
			writeError(w, 404, "aa.not_found", "not found")
		}
	}))
}
