package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeVisualizerFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func rulesOf(violations []BoundaryViolation) map[string]bool {
	rules := map[string]bool{}
	for _, v := range violations {
		rules[v.Rule] = true
	}
	return rules
}

func TestVisualizerBoundariesWorkspace(t *testing.T) {
	root := filepath.Join("..", "..")
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		t.Error(v.String())
	}
}

func TestVisualizerBoundariesAllowSeams(t *testing.T) {
	root := t.TempDir()
	writeVisualizerFile(t, root, "visualizer/adapter/adapter.go", `package adapter
import "github.com/greadee/aa/obsv/protocol"
var _ = protocol.Version
`)
	writeVisualizerFile(t, root, "visualizer/source/source.go", `package source
import (
	"github.com/greadee/aa/obsv/transport"
	"github.com/greadee/aa/obsv/journal"
)
var _ = transport.Server(nil)
var _ = journal.Journal(nil)
`)
	writeVisualizerFile(t, root, "visualizer/source/source_test.go", `package source
import "github.com/greadee/aa/obsv/transport"
var _ = transport.Server(nil)
`)
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestVisualizerBoundariesDetectSecondProtocolImport(t *testing.T) {
	root := t.TempDir()
	writeVisualizerFile(t, root, "visualizer/adapter/adapter.go", `package adapter
import "github.com/greadee/aa/obsv/protocol"
var _ = protocol.Version
`)
	writeVisualizerFile(t, root, "visualizer/sneaky/sneaky.go", `package sneaky
import "github.com/greadee/aa/obsv/protocol"
var _ = protocol.Version
`)
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rulesOf(violations)[RuleSingleAdapter] {
		t.Fatalf("expected %s, got %v", RuleSingleAdapter, violations)
	}
}

func TestVisualizerBoundariesDetectSecondTransportImport(t *testing.T) {
	root := t.TempDir()
	writeVisualizerFile(t, root, "visualizer/source/source.go", `package source
import "github.com/greadee/aa/obsv/transport"
var _ = transport.Server(nil)
`)
	writeVisualizerFile(t, root, "visualizer/sneaky/sneaky.go", `package sneaky
import "github.com/greadee/aa/obsv/journal"
var _ = journal.Journal(nil)
`)
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rulesOf(violations)[RuleSingleTransport] {
		t.Fatalf("expected %s, got %v", RuleSingleTransport, violations)
	}
}

func TestVisualizerBoundariesDetectDuplicateTransport(t *testing.T) {
	root := t.TempDir()
	writeVisualizerFile(t, root, "visualizer/sneaky/transport.go", `package sneaky
type Transport interface {
	Hello() error
	Append() error
	Replay() error
	Subscribe() error
}
`)
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rulesOf(violations)[RuleNoTransport] {
		t.Fatalf("expected %s, got %v", RuleNoTransport, violations)
	}
}

func TestVisualizerBoundariesDetectDuplicateEventStoreStruct(t *testing.T) {
	root := t.TempDir()
	writeVisualizerFile(t, root, "visualizer/sneaky/store.go", `package sneaky
type Store struct{}

func (s Store) Append() error    { return nil }
func (s Store) Replay() error    { return nil }
func (s Store) Subscribe() error { return nil }
func (s Store) Trim() error      { return nil }
func (s Store) Len() int         { return 0 }
`)
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rulesOf(violations)[RuleNoTransport] {
		t.Fatalf("expected %s, got %v", RuleNoTransport, violations)
	}
}

func TestVisualizerBoundariesDetectDuplicateProtocol(t *testing.T) {
	root := t.TempDir()
	writeVisualizerFile(t, root, "visualizer/sneaky/event.go", `package sneaky
type Observation struct {
	SessionID  string
	Sequence   int
	Confidence string
}
`)
	violations, err := CheckVisualizerBoundaries(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rulesOf(violations)[RuleNoProtocol] {
		t.Fatalf("expected %s, got %v", RuleNoProtocol, violations)
	}
}

func TestVisualizerBoundaryViolationString(t *testing.T) {
	v := BoundaryViolation{Rule: RuleNoProtocol, File: "visualizer/x.go", Detail: "detail"}
	if got := v.String(); !strings.Contains(got, RuleNoProtocol) || !strings.Contains(got, "detail") {
		t.Fatalf("String() = %q", got)
	}
}
