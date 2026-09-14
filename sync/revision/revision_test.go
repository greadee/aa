package revision

import (
	"errors"
	"testing"

	aasync "github.com/greadee/aa/sync"
)

func rec(path string, rev uint64, hash string, deleted bool) aasync.RevisionRecord {
	return aasync.RevisionRecord{ID: path, Path: path, Revision: rev, Hash: hash, Deleted: deleted}
}

func TestSetPutRejectsOlder(t *testing.T) {
	s := NewSet()
	if err := s.Put(rec("a", 3, "h3", false)); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := s.Put(rec("a", 2, "h2", false)); !errors.Is(err, aasync.ErrConflict) {
		t.Fatalf("older put err = %v, want ErrConflict", err)
	}
	if err := s.Put(aasync.RevisionRecord{Path: ""}); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("empty path err = %v, want ErrInvalid", err)
	}
}

func TestDigestIsOrderIndependent(t *testing.T) {
	a := NewSet()
	_ = a.Put(rec("b", 1, "hb", false))
	_ = a.Put(rec("a", 1, "ha", false))
	b := NewSet()
	_ = b.Put(rec("a", 1, "ha", false))
	_ = b.Put(rec("b", 1, "hb", false))
	if a.Digest() != b.Digest() {
		t.Fatal("digest depends on insertion order")
	}
}

func TestDiffAndApplyConverge(t *testing.T) {
	source := NewSet()
	_ = source.Put(rec("t1", 1, "h1", false))
	_ = source.Put(rec("t2", 2, "h2", false))
	_ = source.Put(rec("t3", 3, "h3", true))

	replica := NewSet()
	_ = replica.Put(rec("t1", 1, "h1", false))
	_ = replica.Put(rec("t2", 1, "h1", false))
	_ = replica.Put(rec("t3", 2, "h2", false))

	ops := Diff(source, replica)
	if len(ops) != 2 {
		t.Fatalf("ops = %+v, want 2", ops)
	}
	if ops[0].Kind != OpPut || ops[0].Record.Path != "t2" {
		t.Fatalf("op0 = %+v", ops[0])
	}
	if ops[1].Kind != OpDelete || ops[1].Record.Path != "t3" {
		t.Fatalf("op1 = %+v", ops[1])
	}

	res, err := Apply(replica, ops)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 2 || len(res.Refused) != 0 {
		t.Fatalf("result = %+v", res)
	}
	if replica.Digest() != source.Digest() {
		t.Fatal("replicas did not converge")
	}

	// Applying again is a no-op.
	res, _ = Apply(replica, ops)
	if res.Applied != 0 {
		t.Fatalf("re-apply applied %d, want 0", res.Applied)
	}
}

func TestDiffIgnoresAbsentSourcePaths(t *testing.T) {
	source := NewSet()
	replica := NewSet()
	_ = replica.Put(rec("keep", 1, "h", false))
	if ops := Diff(source, replica); len(ops) != 0 {
		t.Fatalf("ops = %+v, want none (absence is not deletion)", ops)
	}
}

func TestDeletionGuardRefusesNewerLocal(t *testing.T) {
	replica := NewSet()
	_ = replica.Put(rec("guarded", 5, "h5", false))
	tombstone := aasync.Op{Kind: aasync.OpDelete, Record: rec("guarded", 3, "h3", true)}
	res, err := Apply(replica, []aasync.Op{tombstone})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(res.Refused) != 1 || res.Applied != 0 {
		t.Fatalf("result = %+v, want one refusal", res)
	}
	if got, _ := replica.Get("guarded"); got.Deleted {
		t.Fatal("guarded path was deleted")
	}
}
