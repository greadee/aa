package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/greadee/aa/ui/controlplane"
)

func TestWebReadOnly(t *testing.T) {
	srv := httptest.NewServer(New(controlplane.NewFake(), "default").Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "aa console") || !strings.Contains(string(body), "Default") {
		t.Fatalf("page = %d\n%s", resp.StatusCode, body)
	}

	resp, err = http.Get(srv.URL + "/plain")
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(plain), "view:") {
		t.Fatalf("plain =\n%s", plain)
	}
}
