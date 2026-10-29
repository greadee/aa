package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/repo"
)

func TestProjectScopedReadRoutes(t *testing.T) {
	root := t.TempDir()
	g, srv := startControlPlane(t, root)
	defer stopControlPlane(t, g, srv)

	for _, body := range []string{
		`{"id":"p1","name":"One"}`,
		`{"id":"p2","name":"Two"}`,
	} {
		if status, resp := apiRequest(t, srv, http.MethodPost, "/api/v1/projects", body); status != http.StatusCreated {
			t.Fatalf("create project = %d %s", status, resp)
		}
	}

	issues := repo.NewIssueRepository(g.store, g.projection)
	for _, project := range []string{"p1", "p2"} {
		if _, err := issues.Create(v2.Issue{
			Envelope: v2.Envelope{ProjectID: project},
			Title:    "issue in " + project,
			Type:     "bug",
		}); err != nil {
			t.Fatalf("create issue for %s: %v", project, err)
		}
	}

	strategies := repo.NewStrategyRepository(g.store, g.projection)
	for _, project := range []string{"p1", "p2"} {
		if _, err := strategies.Propose(v2.Strategy{
			Envelope: v2.Envelope{ProjectID: project},
			Title:    "strategy in " + project,
			Guidance: "use " + project,
		}); err != nil {
			t.Fatalf("create strategy for %s: %v", project, err)
		}
	}

	for i, project := range []string{"p1", "p2"} {
		event := []byte(`{"contractVersion":"2.0","kind":"event","id":"evt_` + project + `","projectId":"` + project + `","sequence":` + string(rune('1'+i)) + `,"occurredAt":"2026-01-01T00:00:00Z","type":"TELEMETRY_RECORDED","aggregate":{"kind":"project","id":"` + project + `"},"actor":{"kind":"agent","id":"a1"},"payload":{"label":"event.` + project + `"}}`)
		if _, err := g.store.AppendEvent(event); err != nil {
			t.Fatalf("append event for %s: %v", project, err)
		}
	}

	cases := []struct {
		path      string
		wantLabel string
		badLabel  string
	}{
		{"/api/v1/projects/p1/issues", "issue in p1", "issue in p2"},
		{"/api/v1/projects/p1/strategies", "strategy in p1", "strategy in p2"},
		{"/api/v1/projects/p1/events", "event.p1", "event.p2"},
	}
	for _, c := range cases {
		status, body := apiRequest(t, srv, http.MethodGet, c.path, "")
		if status != http.StatusOK {
			t.Fatalf("%s = %d %s", c.path, status, body)
		}
		var items []map[string]any
		if err := json.Unmarshal(body, &items); err != nil {
			t.Fatalf("%s decode: %v (%s)", c.path, err, body)
		}
		if len(items) == 0 {
			t.Fatalf("%s returned no records", c.path)
		}
		raw := string(body)
		if !strings.Contains(raw, c.wantLabel) {
			t.Fatalf("%s body = %s, want %q", c.path, raw, c.wantLabel)
		}
		if strings.Contains(raw, c.badLabel) {
			t.Fatalf("%s leaked another project: %s", c.path, raw)
		}
	}
}
