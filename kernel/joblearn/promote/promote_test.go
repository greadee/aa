package promote

import (
	"errors"
	"testing"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/memory/repo"
)

var (
	errNotCandidate = errors.New("fake sink: not a candidate")
	errMissing      = errors.New("fake sink: missing record")
	errIllegal      = errors.New("fake sink: illegal transition")
)

type fakeSink struct {
	records   map[string]v1.MemoryRecord
	proposals int
}

func newFakeSink() *fakeSink { return &fakeSink{records: map[string]v1.MemoryRecord{}} }

func (f *fakeSink) Propose(r v1.MemoryRecord) (v1.MemoryRecord, error) {
	if r.Lifecycle != v1.MemoryCandidate {
		return v1.MemoryRecord{}, errNotCandidate
	}
	f.proposals++
	if existing, ok := f.records[r.ID]; ok {
		return existing, nil
	}
	f.records[r.ID] = r
	return r, nil
}

func (f *fakeSink) Transition(id string, to v1.MemoryLifecycle) (v1.MemoryRecord, error) {
	r, ok := f.records[id]
	if !ok {
		return v1.MemoryRecord{}, errMissing
	}
	if !repo.CanTransition(r.Lifecycle, to) {
		return v1.MemoryRecord{}, errIllegal
	}
	r.Lifecycle = to
	f.records[id] = r
	return r, nil
}

func fixedNow() time.Time { return time.Unix(0, 0).UTC() }

func candidate() joblearn.Candidate {
	return joblearn.Candidate{
		Kind:          joblearn.CandidateStrategy,
		Level:         joblearn.LevelProject,
		Scope:         "work_package:wp1",
		Title:         "Strategy from work package wp1",
		Content:       "3 of 3 attributed outcomes succeeded",
		Applicability: &joblearn.Applicability{Roles: []string{"builder"}},
		Evidence:      []joblearn.Reference{{Kind: "telemetry", ID: "tel_1"}},
		Provenance:    &joblearn.Provenance{Source: "aa-kernel/joblearn", ProducedAt: "2026-01-01T00:00:00Z"},
		Confidence:    0.8,
	}
}

func newPromoter(t *testing.T, sink Sink) *Promoter {
	t.Helper()
	p, err := New(sink, Options{ProjectID: "prj_1", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProposeMapsCandidateToCandidateRecord(t *testing.T) {
	sink := newFakeSink()
	p := newPromoter(t, sink)
	stored, err := p.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored = %d, want 1", len(stored))
	}
	rec := stored[0]
	if rec.Kind != "memory_record" || rec.ContractVersion != v1.Version {
		t.Fatalf("envelope = %+v", rec.Envelope)
	}
	if rec.Lifecycle != v1.MemoryCandidate {
		t.Fatalf("lifecycle = %s, want CANDIDATE", rec.Lifecycle)
	}
	if rec.Level != "project" || rec.Title != candidate().Title || rec.Content.Summary != candidate().Content {
		t.Fatalf("mapping = %+v", rec)
	}
	if rec.ProjectID != "prj_1" {
		t.Fatalf("projectId = %s", rec.ProjectID)
	}
	if rec.Confidence == nil || *rec.Confidence != 0.8 {
		t.Fatalf("confidence = %v", rec.Confidence)
	}
	if rec.Applicability == nil || len(rec.Applicability.Roles) != 1 {
		t.Fatalf("applicability = %+v", rec.Applicability)
	}
	if len(rec.Evidence) != 1 || rec.Evidence[0].ID != "tel_1" {
		t.Fatalf("evidence = %+v", rec.Evidence)
	}
	if rec.Provenance == nil || rec.Provenance.Source != "aa-kernel-joblearn" {
		t.Fatalf("provenance = %+v", rec.Provenance)
	}
	if rec.CreatedAt != "1970-01-01T00:00:00Z" {
		t.Fatalf("createdAt = %s", rec.CreatedAt)
	}
}

func TestProposeIsIdempotent(t *testing.T) {
	sink := newFakeSink()
	p := newPromoter(t, sink)
	first, err := p.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	if first[0].ID != second[0].ID {
		t.Fatalf("ids differ: %s vs %s", first[0].ID, second[0].ID)
	}
	if len(sink.records) != 1 {
		t.Fatalf("records = %d, want 1", len(sink.records))
	}
}

func TestProposeDeduplicatesWithinBatch(t *testing.T) {
	sink := newFakeSink()
	p := newPromoter(t, sink)
	stored, err := p.Propose([]joblearn.Candidate{candidate(), candidate()})
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored = %d, want 1", len(stored))
	}
}

func TestIdentityIsDeterministic(t *testing.T) {
	a := newPromoter(t, newFakeSink())
	b := newPromoter(t, newFakeSink())
	first, err := a.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	if first[0].ID != second[0].ID {
		t.Fatalf("ids differ across promoters: %s vs %s", first[0].ID, second[0].ID)
	}
}

func TestProposeRejectsInvalidCandidate(t *testing.T) {
	p := newPromoter(t, newFakeSink())
	if _, err := p.Propose([]joblearn.Candidate{{Level: joblearn.LevelProject, Title: "t", Content: "c"}}); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestProposeNeverPromotes(t *testing.T) {
	sink := newFakeSink()
	p := newPromoter(t, sink)
	stored, err := p.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	if sink.records[stored[0].ID].Lifecycle != v1.MemoryCandidate {
		t.Fatal("proposal auto-promoted")
	}
}

func TestPromoteIsExplicitAndValidated(t *testing.T) {
	sink := newFakeSink()
	p := newPromoter(t, sink)
	stored, err := p.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	id := stored[0].ID
	if _, err := p.Promote(id, v1.MemoryActive); !errors.Is(err, errIllegal) {
		t.Fatalf("CANDIDATE -> ACTIVE err = %v, want illegal", err)
	}
	if _, err := p.Promote(id, v1.MemoryValidated); err != nil {
		t.Fatal(err)
	}
	active, err := p.Promote(id, v1.MemoryActive)
	if err != nil {
		t.Fatal(err)
	}
	if active.Lifecycle != v1.MemoryActive {
		t.Fatalf("lifecycle = %s", active.Lifecycle)
	}
	if _, err := p.Promote("", v1.MemoryValidated); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("empty id err = %v, want ErrInvalid", err)
	}
	if _, err := p.Promote(id, v1.MemoryLifecycle("BOGUS")); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("bad lifecycle err = %v, want ErrInvalid", err)
	}
}

func TestNewRequiresSink(t *testing.T) {
	if _, err := New(nil, Options{}); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestNewMemorySinkRequiresRepository(t *testing.T) {
	if _, err := NewMemorySink(nil); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestNormalizeSource(t *testing.T) {
	cases := map[string]string{
		"":                   DefaultSource,
		"aa-kernel.joblearn": "aa-kernel.joblearn",
		"aa-kernel/joblearn": "aa-kernel-joblearn",
		"/leading":           "src-leading",
		"has spaces":         "has-spaces",
		"!!":                 "src--",
	}
	for in, want := range cases {
		if got := normalizeSource(in); got != want {
			t.Errorf("normalizeSource(%q) = %q, want %q", in, got, want)
		}
	}
}
