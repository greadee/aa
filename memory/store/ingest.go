package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// IngestOptions controls ingestion policy.
type IngestOptions struct {
	// RequireProvenance rejects records that do not carry a provenance block.
	RequireProvenance bool
}

// IngestResult reports ingestion counts.
type IngestResult struct {
	New       int
	Unchanged int
}

// IngestRecords ingests records idempotently. Re-ingesting identical content
// is a no-op and is counted as unchanged.
func (s *Store) IngestRecords(records []Record, opts IngestOptions) (IngestResult, error) {
	var res IngestResult
	for _, rec := range records {
		if opts.RequireProvenance {
			if err := RequireProvenance(rec.Data); err != nil {
				return res, fmt.Errorf("record %s/%s: %w", rec.Kind, rec.ID, err)
			}
		}
		changed, err := s.PutRecord(rec)
		if err != nil {
			return res, err
		}
		if changed {
			res.New++
		} else {
			res.Unchanged++
		}
	}
	return res, nil
}

// IngestEvent ingests an event idempotently.
func (s *Store) IngestEvent(event []byte) (IngestResult, error) {
	changed, err := s.AppendEvent(event)
	if err != nil {
		return IngestResult{}, err
	}
	if changed {
		return IngestResult{New: 1}, nil
	}
	return IngestResult{Unchanged: 1}, nil
}

// provenance is the minimal provenance shape.
type provenance struct {
	Source     string `json:"source"`
	ProducedAt string `json:"producedAt"`
}

// HasProvenance reports whether data carries a provenance block.
func HasProvenance(data []byte) bool {
	var wrapper struct {
		Provenance *provenance `json:"provenance"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return false
	}
	return wrapper.Provenance != nil
}

// RequireProvenance requires and validates the provenance block.
func RequireProvenance(data []byte) error {
	var wrapper struct {
		Provenance *provenance `json:"provenance"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	if wrapper.Provenance == nil {
		return fmt.Errorf("provenance: is required")
	}
	if wrapper.Provenance.Source == "" {
		return fmt.Errorf("provenance.source: is required")
	}
	if _, err := time.Parse(time.RFC3339, wrapper.Provenance.ProducedAt); err != nil {
		return fmt.Errorf("provenance.producedAt: %w", err)
	}
	return nil
}

// ValidateProvenance validates provenance when present and is a no-op otherwise.
func ValidateProvenance(data []byte) error {
	if !HasProvenance(data) {
		return nil
	}
	return RequireProvenance(data)
}
