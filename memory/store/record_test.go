package store

import "testing"

func TestCanonicalizeIsKeyOrderIndependent(t *testing.T) {
	a, err := Canonicalize([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Canonicalize([]byte(`{"a":1,"b":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("canonical forms differ: %s vs %s", a, b)
	}
	ha, _ := HashBytes(a)
	hb, _ := HashBytes(b)
	if ha != hb {
		t.Fatalf("hashes differ: %s vs %s", ha, hb)
	}
}

func TestNewRecordComputesHashAndValidates(t *testing.T) {
	rec, err := NewRecord("issue", "iss_1", Revision, []byte(`{"kind":"issue","id":"iss_1","title":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Hash == "" || len(rec.Hash) != 64 {
		t.Fatalf("unexpected hash %q", rec.Hash)
	}
	if err := rec.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	rec.Hash = "deadbeef"
	if err := rec.Validate(); err == nil {
		t.Fatal("expected hash mismatch")
	}
}

func TestNewRecordRejectsBadIdentifiers(t *testing.T) {
	cases := []struct{ kind, id string }{
		{"", "x"},
		{"issue", ""},
		{"../issue", "x"},
		{"issue", "../../etc/passwd"},
		{"issue", `a\b`},
	}
	for _, c := range cases {
		if _, err := NewRecord(c.kind, c.id, 1, []byte(`{}`)); err == nil {
			t.Errorf("expected error for kind=%q id=%q", c.kind, c.id)
		}
	}
}

func TestRevisionConstant(t *testing.T) {
	if Revision != 1 {
		t.Fatalf("Revision = %d, want 1", Revision)
	}
}
