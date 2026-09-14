package queue

import (
	"errors"
	"testing"

	aasync "github.com/greadee/aa/sync"
)

func ch(node string, path string, rev uint64, hash string) aasync.Change {
	return aasync.Change{
		Node:   aasync.NodeID(node),
		Record: aasync.RevisionRecord{ID: path, Path: path, Revision: rev, Hash: hash},
	}
}

func TestQueueKeepsNewestPerPath(t *testing.T) {
	q := New()
	if err := q.Enqueue(ch("n1", "a", 1, "h1")); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Enqueue(ch("n1", "a", 3, "h3")); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := q.Enqueue(ch("n1", "a", 2, "h2")); err != nil {
		t.Fatalf("enqueue older: %v", err)
	}
	if q.Len() != 1 {
		t.Fatalf("len = %d, want 1", q.Len())
	}
	pending := q.Pending()
	if len(pending) != 1 || pending[0].Record.Revision != 3 {
		t.Fatalf("pending = %+v", pending)
	}
	if err := q.Enqueue(aasync.Change{}); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("empty path err = %v, want ErrInvalid", err)
	}
	if drained := q.Drain(); len(drained) != 1 || q.Len() != 0 {
		t.Fatalf("drain = %+v, len = %d", drained, q.Len())
	}
}

func TestReconcileDecisions(t *testing.T) {
	local := []aasync.Change{
		ch("n1", "a", 2, "A2"),
		ch("n1", "b", 1, "B1"),
		ch("n1", "c", 2, "C"),
		ch("n1", "d", 2, "D-local"),
	}
	remote := []aasync.Change{
		ch("n2", "a", 1, "A1"),
		ch("n2", "b", 3, "B3"),
		ch("n2", "c", 2, "C"),
		ch("n2", "d", 2, "D-remote"),
	}
	d, err := Reconcile(local, remote)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(d.Send) != 1 || d.Send[0].Record.Path != "a" {
		t.Fatalf("send = %+v", d.Send)
	}
	if len(d.Apply) != 1 || d.Apply[0].Record.Path != "b" {
		t.Fatalf("apply = %+v", d.Apply)
	}
	if len(d.Converged) != 1 || d.Converged[0] != "c" {
		t.Fatalf("converged = %+v", d.Converged)
	}
	if len(d.Conflicts) != 1 || d.Conflicts[0].Path != "d" {
		t.Fatalf("conflicts = %+v", d.Conflicts)
	}
}

func TestReconcileDisjointPaths(t *testing.T) {
	local := []aasync.Change{ch("n1", "only-local", 1, "L")}
	remote := []aasync.Change{ch("n2", "only-remote", 1, "R")}
	d, err := Reconcile(local, remote)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(d.Send) != 1 || len(d.Apply) != 1 {
		t.Fatalf("decision = %+v", d)
	}
}
