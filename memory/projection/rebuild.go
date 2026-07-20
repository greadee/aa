package projection

import "github.com/greadee/aa/memory/store"

// Rebuild discards the projection and reconstructs it from canonical records.
// Rebuilding twice from the same store yields the same digest.
func Rebuild(p Projection, s *store.Store) error {
	records, err := s.ListAllRecords()
	if err != nil {
		return err
	}
	p.Reset()
	for _, r := range records {
		if err := p.Put(FromRecord(r)); err != nil {
			return err
		}
	}
	return nil
}
