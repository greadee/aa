package template

import (
	"strings"
	"testing"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	root, ok := FindRoot(".")
	if !ok {
		t.Fatal("template root not found from the package directory")
	}
	engine, err := NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return engine
}

func TestNamesLoadsProcessTemplates(t *testing.T) {
	engine := newTestEngine(t)
	names := strings.Join(engine.Names(), ",")
	for _, want := range []string{"github/feature-issue.md", "github/audit-finding.md", "github/phase-pr.md", "phase/plan.md"} {
		if !strings.Contains(names, want) {
			t.Errorf("missing template %q in %s", want, names)
		}
	}
}

func TestRenderFeatureIssue(t *testing.T) {
	engine := newTestEngine(t)
	out, err := engine.Render("github/feature-issue.md", map[string]any{
		"Title":        "Add forge templates",
		"Goal":         "Render the process docs.",
		"Requirements": []string{"one", "two"},
		"Acceptance":   []string{"renders"},
		"Notes":        "note",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"# Add forge templates", "Render the process docs.", "- one", "- two", "- [ ] renders", "note"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderAuditFinding(t *testing.T) {
	engine := newTestEngine(t)
	out, err := engine.Render("github/audit-finding.md", map[string]any{
		"Title":          "Leaked token",
		"Finding":        "A token was committed.",
		"Classification": "security",
		"Severity":       "P0",
		"Evidence":       "commit abc",
		"Impact":         "high",
		"Recommendation": "rotate",
		"Scope":          "repo",
		"SuggestedPhase": "ph11-release",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "# Leaked token") || !strings.Contains(out, "P0") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	engine := newTestEngine(t)
	data := map[string]any{"Title": "T", "Goal": "G"}
	first, _ := engine.Render("github/feature-issue.md", data)
	second, _ := engine.Render("github/feature-issue.md", data)
	if first != second {
		t.Fatal("rendering is not deterministic")
	}
}

func TestRenderStringAndDefaultFunc(t *testing.T) {
	out, err := RenderString(`{{ default "fallback" .Value }}|{{ upper .Name }}`, map[string]any{"Value": "", "Name": "forge"})
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if out != "fallback|FORGE" {
		t.Fatalf("output = %q", out)
	}
}

func TestDefaultRootRespectsEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DefaultEnv, dir)
	root, ok := DefaultRoot(".")
	if !ok || root != dir {
		t.Fatalf("DefaultRoot = %q, %v", root, ok)
	}
}

func TestNewEngineRejectsMissingRoot(t *testing.T) {
	if _, err := NewEngine("does-not-exist"); err == nil {
		t.Fatal("expected an error for a missing root")
	}
}
