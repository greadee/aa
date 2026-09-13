package v1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validatable is implemented by every contract object that has a Validate method.
type validatable interface {
	Validate() error
}

// decodeByKind decodes a fixture into the Go type named by its "kind".
func decodeByKind(t *testing.T, data []byte) validatable {
	t.Helper()
	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		t.Fatalf("probe kind: %v", err)
	}
	must := func(v validatable, err error) validatable {
		if err != nil {
			t.Fatalf("decode %s: %v", probe.Kind, err)
		}
		return v
	}
	var v validatable
	switch probe.Kind {
	case "event":
		x := &Event{}
		v = must(x, json.Unmarshal(data, x))
	case "work_package":
		x := &WorkPackage{}
		v = must(x, json.Unmarshal(data, x))
	case "task_graph":
		x := &TaskGraph{}
		v = must(x, json.Unmarshal(data, x))
	case "execution_contract":
		x := &ExecutionContract{}
		v = must(x, json.Unmarshal(data, x))
	case "result_envelope":
		x := &ResultEnvelope{}
		v = must(x, json.Unmarshal(data, x))
	case "telemetry":
		x := &Telemetry{}
		v = must(x, json.Unmarshal(data, x))
	case "project_record":
		x := &ProjectRecord{}
		v = must(x, json.Unmarshal(data, x))
	case "memory_record":
		x := &MemoryRecord{}
		v = must(x, json.Unmarshal(data, x))
	case "issue":
		x := &Issue{}
		v = must(x, json.Unmarshal(data, x))
	case "strategy":
		x := &Strategy{}
		v = must(x, json.Unmarshal(data, x))
	case "route_request":
		x := &RouteRequest{}
		v = must(x, json.Unmarshal(data, x))
	case "route_response":
		x := &RouteResponse{}
		v = must(x, json.Unmarshal(data, x))
	case "tool_manifest":
		x := &ToolManifest{}
		v = must(x, json.Unmarshal(data, x))
	case "workflow":
		x := &Workflow{}
		v = must(x, json.Unmarshal(data, x))
	default:
		t.Fatalf("no Go binding for kind %q", probe.Kind)
	}
	return v
}

func fixtures(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no fixtures found")
	}
	return paths
}

// TestValidFixturesDecodeAndValidate proves each binding accepts its fixture.
func TestValidFixturesDecodeAndValidate(t *testing.T) {
	for _, path := range fixtures(t) {
		name := filepath.Base(path)
		if strings.Contains(name, "invalid") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			v := decodeByKind(t, data)
			if err := v.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
		})
	}
}

// TestRoundTripIsStable proves decode -> encode -> decode -> encode is byte-stable.
func TestRoundTripIsStable(t *testing.T) {
	for _, path := range fixtures(t) {
		name := filepath.Base(path)
		if strings.Contains(name, "invalid") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			v := decodeByKind(t, data)
			first, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			second := decodeByKind(t, first)
			again, err := json.Marshal(second)
			if err != nil {
				t.Fatal(err)
			}
			if string(first) != string(again) {
				t.Fatalf("round trip not stable:\n first: %s\nsecond: %s", first, again)
			}
		})
	}
}

// TestInvalidFixturesAreRejected proves validation fails closed.
func TestInvalidFixturesAreRejected(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*invalid*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("expected at least one invalid fixture")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			v := decodeByKind(t, data)
			if err := v.Validate(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

// TestSchemasAreWellFormed proves every schema parses and declares its metadata.
func TestSchemasAreWellFormed(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "schemas", "v1", "*.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no schemas found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Schema      string `json:"$schema"`
				ID          string `json:"$id"`
				ContractVer string `json:"x-contract-version"`
				Title       string `json:"title"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !strings.HasSuffix(doc.Schema, "2020-12/schema") {
				t.Errorf("$schema: got %q", doc.Schema)
			}
			if !strings.HasPrefix(doc.ID, "https://github.com/greadee/aa/contracts/schemas/v1/") {
				t.Errorf("$id: got %q", doc.ID)
			}
			if doc.ContractVer == "" {
				t.Error("x-contract-version is required")
			}
			if doc.Title == "" {
				t.Error("title is required")
			}
		})
	}
}

// TestVersion guards the shared contract version.
func TestVersion(t *testing.T) {
	if Version != "1.0" {
		t.Fatalf("Version: got %q, want 1.0", Version)
	}
}
