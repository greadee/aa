package repo

import (
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// StrategyRepository stores and promotes canonical strategies.
type StrategyRepository struct {
	store *store.Store
	proj  projection.Projection
	newID idgen
}

// NewStrategyRepository returns a strategy repository over the store and projection.
func NewStrategyRepository(s *store.Store, p projection.Projection) *StrategyRepository {
	return &StrategyRepository{store: s, proj: p, newID: func() string { return "str_" + randomSuffix() }}
}

// Propose stores a new strategy in the CANDIDATE state.
func (r *StrategyRepository) Propose(strategy v2.Strategy) (v2.Strategy, error) {
	if strategy.ContractVersion == "" {
		strategy.ContractVersion = v2.Version
	}
	strategy.Kind = "strategy"
	if strategy.ID == "" {
		strategy.ID = r.newID()
	}
	if strategy.Lifecycle == "" {
		strategy.Lifecycle = v2.MemoryCandidate
	}
	if strategy.Lifecycle != v2.MemoryCandidate {
		return v2.Strategy{}, fmt.Errorf("strategy: propose requires CANDIDATE lifecycle, got %s", strategy.Lifecycle)
	}
	rev := 1
	strategy.Revision = &rev
	if err := strategy.Validate(); err != nil {
		return v2.Strategy{}, err
	}
	if _, err := putTyped(r.store, r.proj, "strategy", strategy.ID, rev, strategy); err != nil {
		return v2.Strategy{}, err
	}
	return strategy, nil
}

// Get returns a strategy by id.
func (r *StrategyRepository) Get(id string) (v2.Strategy, error) {
	return getTyped[v2.Strategy](r.store, "strategy", id)
}

// List returns all strategies, sorted by id.
func (r *StrategyRepository) List() ([]v2.Strategy, error) {
	records, err := r.store.ListRecords("strategy")
	if err != nil {
		return nil, err
	}
	out := make([]v2.Strategy, 0, len(records))
	for _, rec := range records {
		var strategy v2.Strategy
		if err := decode(rec, &strategy); err != nil {
			return nil, err
		}
		out = append(out, strategy)
	}
	return out, nil
}

// ListActive returns strategies in the ACTIVE lifecycle.
func (r *StrategyRepository) ListActive() ([]v2.Strategy, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	var out []v2.Strategy
	for _, s := range all {
		if s.Lifecycle == v2.MemoryActive {
			out = append(out, s)
		}
	}
	return out, nil
}

// Transition applies a validated lifecycle transition.
func (r *StrategyRepository) Transition(id string, to v2.MemoryLifecycle) (v2.Strategy, error) {
	strategy, err := r.Get(id)
	if err != nil {
		return v2.Strategy{}, err
	}
	if _, err := Transition(strategy.Lifecycle, to); err != nil {
		return v2.Strategy{}, fmt.Errorf("strategy %s: %w", id, err)
	}
	if strategy.Lifecycle == to {
		return strategy, nil
	}
	strategy.Lifecycle = to
	rev := revisionOf(r.store, "strategy", id) + 1
	strategy.Revision = &rev
	if err := strategy.Validate(); err != nil {
		return v2.Strategy{}, err
	}
	if _, err := putTyped(r.store, r.proj, "strategy", strategy.ID, rev, strategy); err != nil {
		return v2.Strategy{}, err
	}
	return strategy, nil
}

// Supersede marks oldID superseded and points newID at it.
func (r *StrategyRepository) Supersede(oldID, newID string) error {
	if oldID == newID {
		return fmt.Errorf("strategy: cannot supersede itself")
	}
	if _, err := r.Transition(oldID, v2.MemorySuperseded); err != nil {
		return err
	}
	next, err := r.Get(newID)
	if err != nil {
		return err
	}
	next.Supersedes = oldID
	rev := revisionOf(r.store, "strategy", newID) + 1
	next.Revision = &rev
	if err := next.Validate(); err != nil {
		return err
	}
	_, err = putTyped(r.store, r.proj, "strategy", next.ID, rev, next)
	return err
}
