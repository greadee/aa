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
			_ = json.NewEncoder(w).Encode([]Project{{ID: "default", Name: "Default"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/default/work-packages":
			_ = json.NewEncoder(w).Encode([]WorkPackage{{ID: "wp-1", State: "PROPOSED"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/default/work-packages":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "wp-9"})
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
	st, _ := f.Status(ctx)
	if st.Projects == 0 || st.Nodes == 0 {
		t.Fatalf("status %+v", st)
	}
	if err := f.Approve(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
}
