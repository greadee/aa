package controlplane

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type approvalBackend struct {
	fakeBackend
	gateID, decision, actor, reason string
	err                             error
}

func (b *approvalBackend) Approve(gateID, decision, actor, reason string) error {
	b.gateID, b.decision, b.actor, b.reason = gateID, decision, actor, reason
	return b.err
}

func TestApprovalBodyAndDecision(t *testing.T) {
	backend := &approvalBackend{}
	s := New(Config{Version: "0.1.0", Token: "secret", Status: func() Status { return Status{} }, Backend: backend})
	req := httptest.NewRequest(http.MethodPost, APIPrefix+"/approvals/g1", strings.NewReader(`{"decision":"reject","actor":"human","reason":"nope"}`))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"rejected"`) {
		t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
	}
	if backend.gateID != "g1" || backend.decision != "reject" || backend.actor != "human" || backend.reason != "nope" {
		t.Fatalf("backend = %+v", backend)
	}
}

func TestApprovalRejectsMalformedRequests(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "secret", Status: func() Status { return Status{} }, Backend: &approvalBackend{}})
	cases := []struct {
		body string
		code int
	}{
		{``, http.StatusBadRequest},
		{`{}`, http.StatusBadRequest},
		{`{"decision":"maybe","actor":"human"}`, http.StatusBadRequest},
		{`{"decision":"approve"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, APIPrefix+"/approvals/g1", strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.code {
			t.Fatalf("body %q = %d %s", c.body, rec.Code, rec.Body.String())
		}
	}
}

func TestApprovalMapsBackendErrors(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrConflict, http.StatusConflict},
		{ErrInvalid, http.StatusBadRequest},
	}
	for _, c := range cases {
		s := New(Config{Version: "0.1.0", Token: "secret", Status: func() Status { return Status{} }, Backend: &approvalBackend{err: c.err}})
		req := httptest.NewRequest(http.MethodPost, APIPrefix+"/approvals/g1", strings.NewReader(`{"decision":"approve","actor":"human"}`))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != c.code {
			t.Fatalf("err %v = %d %s", c.err, rec.Code, rec.Body.String())
		}
	}
}
