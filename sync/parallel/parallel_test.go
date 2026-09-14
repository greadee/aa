package parallel

import (
	"errors"
	"testing"

	aasync "github.com/greadee/aa/sync"
)

func TestAssignBalancedAndDeterministic(t *testing.T) {
	c, err := NewCoordinator([]aasync.NodeID{"c", "a", "b", "a"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if got := c.Nodes(); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("nodes = %v", got)
	}
	ids := []string{"wp3", "wp1", "wp5", "wp2", "wp4"}
	first, err := c.Assign(ids)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	second, err := c.Assign([]string{"wp1", "wp2", "wp3", "wp4", "wp5"})
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	if len(first) != 3 {
		t.Fatalf("assignments = %d, want 3", len(first))
	}
	total := 0
	for i := range first {
		total += len(first[i].Packages)
		if len(first[i].Packages) != len(second[i].Packages) {
			t.Fatalf("assignment not input-order independent: %+v vs %+v", first[i], second[i])
		}
	}
	if total != 5 {
		t.Fatalf("placed %d packages, want 5", total)
	}
	if MostLoaded(first)-LeastLoaded(first) > 1 {
		t.Fatalf("imbalanced: %+v", first)
	}
}

func TestAssignRejections(t *testing.T) {
	if _, err := NewCoordinator(nil); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("empty nodes err = %v, want ErrInvalid", err)
	}
	if _, err := NewCoordinator([]aasync.NodeID{""}); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("empty node id err = %v, want ErrInvalid", err)
	}
	c, _ := NewCoordinator([]aasync.NodeID{"a"})
	if _, err := c.Assign([]string{"ok", ""}); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("empty package id err = %v, want ErrInvalid", err)
	}
}

func TestLoadHelpers(t *testing.T) {
	assignments := []Assignment{
		{Node: "a", Packages: []string{"1", "2"}},
		{Node: "b", Packages: []string{"3"}},
	}
	if MostLoaded(assignments) != 2 || LeastLoaded(assignments) != 1 {
		t.Fatalf("loads = %d/%d", MostLoaded(assignments), LeastLoaded(assignments))
	}
}
