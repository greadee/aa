package controlplane

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func newAuthorizedRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer t")
	return req
}

func serve(s *Server, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

var (
	specPathRe     = regexp.MustCompile(`^  (/[^:]+):\s*$`)
	specMethodRe   = regexp.MustCompile(`^    (get|post|put|delete|patch):\s*$`)
	specSchemaRe   = regexp.MustCompile(`^    ([A-Za-z0-9_]+):\s*$`)
	specRequiredRe = regexp.MustCompile(`^\s+required:\s*\[(.*)\]\s*$`)
)

type openAPISpec struct {
	routes   map[string]bool
	required map[string][]string
}

func loadOpenAPISpec(t *testing.T) openAPISpec {
	t.Helper()
	f, err := os.Open("../../../contracts/openapi/control-plane-v1.yaml")
	if err != nil {
		t.Fatalf("open spec: %v", err)
	}
	defer f.Close()

	spec := openAPISpec{routes: map[string]bool{}, required: map[string][]string{}}
	inPaths, inSchemas := false, false
	currentPath, currentSchema := "", ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "paths:" {
			inPaths = true
			continue
		}
		if line == "components:" {
			inPaths = false
			inSchemas = false
			continue
		}
		if line == "  schemas:" {
			inSchemas = true
			continue
		}
		if inPaths {
			if m := specPathRe.FindStringSubmatch(line); m != nil {
				currentPath = m[1]
				continue
			}
			if m := specMethodRe.FindStringSubmatch(line); m != nil && currentPath != "" {
				spec.routes[strings.ToUpper(m[1])+" "+APIPrefix+currentPath] = true
				continue
			}
		}
		if inSchemas {
			if m := specSchemaRe.FindStringSubmatch(line); m != nil {
				currentSchema = m[1]
				continue
			}
			if m := specRequiredRe.FindStringSubmatch(line); m != nil && currentSchema != "" {
				if _, exists := spec.required[currentSchema]; exists {
					continue
				}
				for _, field := range strings.Split(m[1], ",") {
					field = strings.TrimSpace(field)
					if field != "" {
						spec.required[currentSchema] = append(spec.required[currentSchema], field)
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan spec: %v", err)
	}
	if len(spec.routes) == 0 || len(spec.required) == 0 {
		t.Fatal("spec parser produced no routes or schemas")
	}
	return spec
}

func TestOpenAPIRouteSetMatchesServer(t *testing.T) {
	spec := loadOpenAPISpec(t)
	s := New(Config{})
	implemented := map[string]bool{}
	for _, route := range s.Routes() {
		implemented[route] = true
	}
	for route := range spec.routes {
		if !implemented[route] {
			t.Errorf("documented route is not implemented: %s", route)
		}
	}
	for route := range implemented {
		if !spec.routes[route] {
			t.Errorf("implemented route is not documented: %s", route)
		}
	}
}

func TestOpenAPIRepresentativeResponsesMatchRequiredFields(t *testing.T) {
	spec := loadOpenAPISpec(t)
	s := New(Config{Version: "0.1.0", Token: "t", Backend: fakeBackend{}})

	cases := []struct {
		method, path, body, schema string
	}{
		{"GET", APIPrefix + "/projects/p1", "", "ProjectRecord"},
		{"POST", APIPrefix + "/projects/p1/work-packages", `{"title":"first"}`, "WorkPackage"},
		{"GET", APIPrefix + "/projects/p1/graph", "", "TaskGraph"},
	}
	for _, c := range cases {
		req := newAuthorizedRequest(c.method, c.path, c.body)
		rec := serve(s, req)
		if rec.Code < 200 || rec.Code >= 300 {
			t.Fatalf("%s %s = %d %s", c.method, c.path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s %s decode: %v", c.method, c.path, err)
		}
		for _, field := range spec.required[c.schema] {
			if _, ok := body[field]; !ok {
				t.Errorf("%s %s is missing required %s field %q: %s", c.method, c.path, c.schema, field, rec.Body.String())
			}
		}
	}
}

func TestApprovalErrorEnvelopeAndDocumentedStatuses(t *testing.T) {
	s := New(Config{Version: "0.1.0", Token: "t", Backend: fakeBackend{}})
	cases := []struct {
		name, body string
		want       int
	}{
		{"malformed body", `{`, 400},
		{"invalid decision", `{"decision":"maybe","actor":"human"}`, 400},
		{"missing actor", `{"decision":"approve","actor":""}`, 400},
	}
	for _, c := range cases {
		rec := serve(s, newAuthorizedRequest("POST", APIPrefix+"/approvals/g1", c.body))
		if rec.Code != c.want {
			t.Fatalf("%s = %d, want %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
		}
		var body struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s decode: %v", c.name, err)
		}
		if body.Error.Code == "" || body.Error.Message == "" {
			t.Fatalf("%s error envelope = %s", c.name, rec.Body.String())
		}
	}
}
