package v2

import "testing"

func TestMemoryRecordAllowsNilConfidence(t *testing.T) {
	record := MemoryRecord{
		Envelope:  Envelope{ContractVersion: Version, Kind: "memory_record", ID: "mem_optional_confidence"},
		Level:     "project",
		Lifecycle: MemoryValidated,
		Content:   MemoryContent{Summary: "Confidence is optional behavior."},
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestMemoryRecordRejectsOutOfRangeConfidence(t *testing.T) {
	bad := 1.5
	record := MemoryRecord{
		Envelope:   Envelope{ContractVersion: Version, Kind: "memory_record", ID: "mem_bad_confidence"},
		Level:      "project",
		Lifecycle:  MemoryValidated,
		Content:    MemoryContent{Summary: "Confidence must be bounded."},
		Confidence: &bad,
	}
	if err := record.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
}
