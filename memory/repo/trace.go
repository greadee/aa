package repo

import (
	"sort"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// TraceRepository stores canonical trace evidence.
//
// Traces are L2 evidence, not promoted knowledge: they are stored as canonical
// records of kind "trace" and never enter the memory lifecycle. Ingestion is
// idempotent by content hash, and a higher revision supersedes a lower one.
type TraceRepository struct {
	store *store.Store
	proj  projection.Projection
	newID idgen
}

// NewTraceRepository returns a trace repository over the store and projection.
func NewTraceRepository(s *store.Store, p projection.Projection) *TraceRepository {
	return &TraceRepository{store: s, proj: p, newID: func() string { return "trc_" + randomSuffix() }}
}

// Ingest validates and stores a trace. It assigns an id and revision when
// absent. The second result reports whether the store changed; re-ingesting
// identical content is a no-op.
func (r *TraceRepository) Ingest(trace v1.Trace) (v1.Trace, bool, error) {
	trace.Kind = "trace"
	if trace.ID == "" {
		trace.ID = r.newID()
	}
	if trace.Revision == nil {
		rev := store.Revision
		trace.Revision = &rev
	}
	if err := trace.Validate(); err != nil {
		return v1.Trace{}, false, err
	}
	changed, err := putTyped(r.store, r.proj, "trace", trace.ID, *trace.Revision, trace)
	if err != nil {
		return v1.Trace{}, false, err
	}
	return trace, changed, nil
}

// Get returns a trace by id.
func (r *TraceRepository) Get(id string) (v1.Trace, error) {
	return getTyped[v1.Trace](r.store, "trace", id)
}

// List returns all traces, sorted by id.
func (r *TraceRepository) List() ([]v1.Trace, error) {
	records, err := r.store.ListRecords("trace")
	if err != nil {
		return nil, err
	}
	out := make([]v1.Trace, 0, len(records))
	for _, rec := range records {
		var trace v1.Trace
		if err := decode(rec, &trace); err != nil {
			return nil, err
		}
		out = append(out, trace)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ListByAttempt returns traces for one attempt, sorted by id.
func (r *TraceRepository) ListByAttempt(attemptID string) ([]v1.Trace, error) {
	if err := v1.RequireIdentifier("attemptId", attemptID); err != nil {
		return nil, err
	}
	return r.filter(func(t v1.Trace) bool { return t.AttemptID == attemptID })
}

// ListByProject returns traces for one project, sorted by id.
func (r *TraceRepository) ListByProject(projectID string) ([]v1.Trace, error) {
	if err := v1.RequireIdentifier("projectId", projectID); err != nil {
		return nil, err
	}
	return r.filter(func(t v1.Trace) bool { return t.ProjectID == projectID })
}

func (r *TraceRepository) filter(keep func(v1.Trace) bool) ([]v1.Trace, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	var out []v1.Trace
	for _, t := range all {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out, nil
}
