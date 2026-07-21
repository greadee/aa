package store

import (
	"encoding/json"
	"fmt"
	"os"
)

type retention struct {
	Floor int `json:"floor"`
}

// Retention returns the event retention floor. Zero means no retention.
func (s *Store) Retention() (int, error) {
	data, err := os.ReadFile(s.layout.RetentionPath())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var r retention
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, fmt.Errorf("store: corrupt retention file: %w", err)
	}
	if r.Floor < 0 {
		return 0, fmt.Errorf("store: invalid retention floor %d", r.Floor)
	}
	return r.Floor, nil
}

// RetainAfter sets the retention floor. The floor is monotonic: it may only
// increase. Events with sequence <= floor are excluded from Events.
func (s *Store) RetainAfter(sequence int) error {
	if sequence < 0 {
		return fmt.Errorf("retention: floor must be >= 0")
	}
	current, err := s.Retention()
	if err != nil {
		return err
	}
	if sequence < current {
		return fmt.Errorf("retention: floor may only increase (have %d, got %d)", current, sequence)
	}
	if sequence == current {
		return nil
	}
	data, err := json.Marshal(retention{Floor: sequence})
	if err != nil {
		return err
	}
	return writeFileAtomic(s.layout.RetentionPath(), data, 0o600)
}
