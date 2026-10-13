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
}
