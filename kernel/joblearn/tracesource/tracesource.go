// Package tracesource is the pure seam through which joblearn consumes bounded
// per-step traces.
//
// The engine never reads the store or any I/O directly: it asks a Source for
// traces and works on plain values. Sources are deterministic and return
// traces in a stable order. The store-backed adapter lives beside the engine
// (memory.go), so the engine package stays free of the canonical store.
package tracesource

import (
	"fmt"
	"sort"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
)

// Source yields bounded traces for learning. Implementations are pure with
// respect to the caller: they must not mutate their input and must return
// traces in a deterministic order.
type Source interface {
	Traces() ([]v1.Trace, error)
}

// SourceFunc adapts a function to a Source.
type SourceFunc func() ([]v1.Trace, error)

// Traces implements Source.
func (f SourceFunc) Traces() ([]v1.Trace, error) { return f() }

// Bounds limit how much evidence is consumed in one load. A non-positive bound
// is unlimited.
type Bounds struct {
	MaxTraces int `json:"maxTraces"`
	MaxSteps  int `json:"maxSteps"`
}

// DefaultBounds is a conservative learning bound.
func DefaultBounds() Bounds {
	return Bounds{MaxTraces: 1000, MaxSteps: 100000}
}

// Validate checks that the bounds are non-negative.
func (b Bounds) Validate() error {
	if b.MaxTraces < 0 || b.MaxSteps < 0 {
		return invalid("bounds must be >= 0")
	}
	return nil
}

// Static is a deterministic in-memory Source. It sorts by trace id on
// construction so the order never depends on how it was built.
type Static struct {
	traces []v1.Trace
}

// NewStatic returns a Source over a copy of traces, sorted by id.
func NewStatic(traces []v1.Trace) *Static {
	ordered := append([]v1.Trace(nil), traces...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	return &Static{traces: ordered}
}

// Traces returns a copy of the traces in stable order.
func (s *Static) Traces() ([]v1.Trace, error) {
	return append([]v1.Trace(nil), s.traces...), nil
}

// Fake is a scripted Source for tests. It returns a fixed set of traces or a
// fixed error, and records how many times it was read.
type Fake struct {
	Traces_  []v1.Trace
	Err      error
	Reads    int
	Validate bool
}

// NewFake returns a fake Source over traces.
func NewFake(traces ...v1.Trace) *Fake {
	return &Fake{Traces_: append([]v1.Trace(nil), traces...)}
}

// Traces implements Source.
func (f *Fake) Traces() ([]v1.Trace, error) {
	f.Reads++
	if f.Err != nil {
		return nil, f.Err
	}
	return append([]v1.Trace(nil), f.Traces_...), nil
}

// Bound returns a bounded copy of traces. It preserves the given order and
// stops adding whole traces once the trace or step bound would be exceeded, so
// a truncated result never contains half a trace.
func Bound(traces []v1.Trace, b Bounds) ([]v1.Trace, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	out := make([]v1.Trace, 0, len(traces))
	steps := 0
	for _, trace := range traces {
		if b.MaxTraces > 0 && len(out) >= b.MaxTraces {
			break
		}
		if b.MaxSteps > 0 && steps+len(trace.Steps) > b.MaxSteps {
			break
		}
		out = append(out, trace)
		steps += len(trace.Steps)
	}
	return out, nil
}

// Load reads from a Source under bounds and validates every trace before it is
// returned. Both the source and the bound are required.
func Load(src Source, b Bounds) ([]v1.Trace, error) {
	if src == nil {
		return nil, invalid("source is required")
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	traces, err := src.Traces()
	if err != nil {
		return nil, err
	}
	bounded, err := Bound(traces, b)
	if err != nil {
		return nil, err
	}
	for i := range bounded {
		if err := bounded[i].Validate(); err != nil {
			return nil, invalid("traces[%d]: %v", i, err)
		}
	}
	return bounded, nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
