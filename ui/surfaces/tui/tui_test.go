package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/greadee/aa/ui/controlplane"
)

func TestSurfaceNavigationAndCommand(t *testing.T) {
	s := New(controlplane.NewFake(), "default")
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.View(), "aa console") {
		t.Fatalf("header missing:\n%s", s.View())
	}

	s.Handle(ctx, "j") // projects -> graph
	if !strings.Contains(s.View(), "aa console \u203a graph") {
		t.Fatalf("navigation failed:\n%s", s.View())
	}

	s.Handle(ctx, ":submit first task")
	if !strings.Contains(s.View(), "PROPOSED") {
		t.Fatalf("submit failed:\n%s", s.View())
	}

	before := len(s.Audit())
	s.Handle(ctx, ":submit first task") // duplicate key -> no-op
	if len(s.Audit()) != before {
		t.Fatalf("audit idempotency failed: %d -> %d", before, len(s.Audit()))
	}

	if !s.Handle(ctx, "q") {
		t.Fatal("quit not honored")
	}
}

func TestExitAliases(t *testing.T) {
	ctx := context.Background()
	for _, input := range []string{"exit", ":exit"} {
		s := New(controlplane.NewFake(), "default")
		if err := s.Refresh(ctx); err != nil {
			t.Fatal(err)
		}
		if !s.Handle(ctx, input) {
			t.Fatalf("%q did not quit", input)
		}
	}
}

func TestArrowTabAndProjectSwitch(t *testing.T) {
	s := New(controlplane.NewFake(), "default")
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	s.Handle(ctx, "\x1b[B")
	if !strings.Contains(s.View(), "aa console \u203a graph") {
		t.Fatalf("down arrow navigation failed:\n%s", s.View())
	}
	s.Handle(ctx, "\t")
	if !strings.Contains(s.View(), "aa console \u203a memory") {
		t.Fatalf("tab navigation failed:\n%s", s.View())
	}
	s.Handle(ctx, "\x1b[A")
	if !strings.Contains(s.View(), "aa console \u203a graph") {
		t.Fatalf("up arrow navigation failed:\n%s", s.View())
	}

	s.Handle(ctx, ":open projects")
	if err := s.client.CreateProject(ctx, "p2", "Project Two"); err != nil {
		t.Fatal(err)
	}
	s.Handle(ctx, ":project p2")
	if !strings.Contains(s.View(), "WORK (p2)") {
		t.Fatalf("project switch failed:\n%s", s.View())
	}
	s.Handle(ctx, ":project missing")
	if !strings.Contains(s.View(), "error:") {
		t.Fatalf("unknown project should report an error:\n%s", s.View())
	}
	if !strings.Contains(s.View(), "WORK (p2)") {
		t.Fatalf("unknown project should not switch:\n%s", s.View())
	}
}

func TestMemoryAndReportsViews(t *testing.T) {
	s := New(controlplane.NewFake(), "default")
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	s.Handle(ctx, ":open memory")
	if !strings.Contains(s.View(), "issue-demo") || !strings.Contains(s.View(), "strategy-demo") {
		t.Fatalf("memory view missing data:\n%s", s.View())
	}
	s.Handle(ctx, ":open reports")
	if !strings.Contains(s.View(), "session-demo") || !strings.Contains(s.View(), "REPORT") {
		t.Fatalf("reports view missing data:\n%s", s.View())
	}
}

func TestRunUntilEOF(t *testing.T) {
	s := New(controlplane.NewFake(), "default")
	in := strings.NewReader(":open tools\n")
	var out strings.Builder
	if err := s.Run(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "tool_screen") {
		t.Fatalf("tools view not rendered:\n%s", out.String())
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatalf("non-terminal output must not contain ANSI redraw escapes:\n%q", out.String())
	}
}
