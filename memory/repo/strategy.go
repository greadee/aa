package repo

import (
	"fmt"

	v1 "github.com/greadee/aa/contracts/go/v1"
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
func (r *StrategyRepository) Propose(strategy v1.Strategy) (v1.Strategy, error) {
	if strategy.ContractVersion == "" {
		strategy.ContractVersion = v1.Version
	}
	strategy.Kind = "strategy"
	if strategy.ID == "" {
		strategy.ID = r.newID()
	}
	if strategy.Lifecycle == "" {
		strategy.Lifecycle = v1.MemoryCandidate
	}
	if strategy.Lifecycle != v1.MemoryCandidate {
		return v1.Strategy{}, fmt.Errorf("strategy: propose requires CANDIDATE lifecycle, got %s", strategy.Lifecycle)
	}
	rev := 1
	strategy.Revision = &rev
	if err := strategy.Validate(); err != nil {
		return v1.Strategy{}, err
	}
	if _, err := putTyped(r.store, r.proj, "strategy", strategy.ID, rev, strategy); err != nil {
		return v1.Strategy{}, err
	}
	return strategy, nil
}

// Get returns a strategy by id.
func (r *StrategyRepository) Get(id string) (v1.Strategy, error) {
	return getTyped[v1.Strategy](r.store, "strategy", id)
}

// List returns all strategies, sorted by id.
func (r *StrategyRepository) List() ([]v1.Strategy, error) {
	records, err := r.store.ListRecords("strategy")
	if err != nil {
		return nil, err
	}
	out := make([]v1.Strategy, 0, len(records))
	for _, rec := range records {
		var strategy v1.Strategy
		if err := decode(rec, &strategy); err != nil {
			return nil, err
		}
		out = append(out, strategy)
	}
	return out, nil
}

// ListActive returns strategies in the ACTIVE lifecycle.
func (r *StrategyRepository) ListActive() ([]v1.Strategy, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	var out []v1.Strategy
	for _, s := range all {
		if s.Lifecycle == v1.MemoryActive {
			out = append(out, s)
		}
	}
	return out, nil
}

// Transition applies a validated lifecycle transition.
func (r *StrategyRepository) Transition(id string, to v1.MemoryLifecycle) (v1.Strategy, error) {
	strategy, err := r.Get(id)
	if err != nil {
		return v1.Strategy{}, err
	}
	if _, err := Transition(strategy.Lifecycle, to); err != nil {
		return v1.Strategy{}, fmt.Errorf("strategy %s: %w", id, err)
	}
	if strategy.Lifecycle == to {
		return strategy, nil
	}
	strategy.Lifecycle = to
	rev := revisionOf(r.store, "strategy", id) + 1
	strategy.Revision = &rev
	if err := strategy.Validate(); err != nil {
		return v1.Strategy{}, err
	}
	if _, err := putTyped(r.store, r.proj, "strategy", strategy.ID, rev, strategy); err != nil {
		return v1.Strategy{}, err
	}
	return strategy, nil
}

// Supersede marks oldID superseded and points newID at it.
func (r *StrategyRepository) Supersede(oldID, newID string) error {
	if oldID == newID {
		return fmt.Errorf("strategy: cannot supersede itself")
	}
	if _, err := r.Transition(oldID, v1.MemorySuperseded); err != nil {
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
