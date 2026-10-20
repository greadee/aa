package controlplane

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRegistry struct{}

func (fakeRegistry) Teams() []json.RawMessage {
	return []json.RawMessage{json.RawMessage(`{"id":"team_eng","title":"Engineering"}`)}
}

func (fakeRegistry) Team(id string) (json.RawMessage, bool) {
	if id != "team_eng" {
		return nil, false
	}
	return json.RawMessage(`{"id":"team_eng","title":"Engineering"}`), true
}

func (fakeRegistry) AddTeam(spec json.RawMessage) error { return registryAdd(spec) }

func (fakeRegistry) Routines() []json.RawMessage {
	return []json.RawMessage{json.RawMessage(`{"id":"routine_feature","name":"Feature"}`)}
}

func (fakeRegistry) Routine(id string) (json.RawMessage, bool) {
	if id != "routine_feature" {
		return nil, false
	}
	return json.RawMessage(`{"id":"routine_feature","name":"Feature"}`), true
}

func (fakeRegistry) AddRoutine(spec json.RawMessage) error { return registryAdd(spec) }

func (fakeRegistry) Policies() []json.RawMessage {
	return []json.RawMessage{json.RawMessage(`{"id":"policy_budget","policyKind":"budget"}`)}
}

func (fakeRegistry) Policy(id string) (json.RawMessage, bool) {
	if id != "policy_budget" {
		return nil, false
	}
	return json.RawMessage(`{"id":"policy_budget","policyKind":"budget"}`), true
}

func (fakeRegistry) AddPolicy(spec json.RawMessage) error { return registryAdd(spec) }

func registryAdd(spec json.RawMessage) error {
	text := string(spec)
	switch {
	case strings.Contains(text, "duplicate"):
		return fmt.Errorf("%w: %s", ErrDefinitionConflict, text)
	case strings.Contains(text, "invalid"):
		return errors.New("invalid definition")
	default:
		return nil
	}
}

func TestRegistryEndpoints(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Registry: fakeRegistry{}})
	cases := []struct {
		method, path, body string
		want               int
		contains           string
	}{
		{http.MethodGet, APIPrefix + "/teams", "", http.StatusOK, "team_eng"},
		{http.MethodGet, APIPrefix + "/teams/team_eng", "", http.StatusOK, "Engineering"},
		{http.MethodGet, APIPrefix + "/teams/missing", "", http.StatusNotFound, "not found"},
		{http.MethodPost, APIPrefix + "/teams", `{"id":"team_new"}`, http.StatusCreated, "created"},
		{http.MethodPost, APIPrefix + "/teams", `{"id":"duplicate"}`, http.StatusConflict, "already exists"},
		{http.MethodPost, APIPrefix + "/teams", `{"id":"invalid"}`, http.StatusBadRequest, "invalid definition"},
		{http.MethodGet, APIPrefix + "/routines", "", http.StatusOK, "routine_feature"},
		{http.MethodGet, APIPrefix + "/routines/routine_feature", "", http.StatusOK, "Feature"},
		{http.MethodPost, APIPrefix + "/routines", `{"id":"routine_new"}`, http.StatusCreated, "created"},
		{http.MethodGet, APIPrefix + "/policies", "", http.StatusOK, "policy_budget"},
		{http.MethodGet, APIPrefix + "/policies/policy_budget", "", http.StatusOK, "budget"},
		{http.MethodPost, APIPrefix + "/policies", `{"id":"policy_new"}`, http.StatusCreated, "created"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d (%s)", c.method, c.path, rec.Code, c.want, rec.Body.String())
			continue
		}
		if !strings.Contains(rec.Body.String(), c.contains) {
			t.Errorf("%s %s body = %s, want %q", c.method, c.path, rec.Body.String(), c.contains)
		}
	}
}

func TestRegistryRequiresToken(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Registry: fakeRegistry{}})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/teams", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRegistryUnconfigured(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/teams", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}
}
