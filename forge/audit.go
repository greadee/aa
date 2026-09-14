package forge

import "time"

// Operation is one auditable forge action.
type Operation struct {
	Seq       int               `json:"seq"`
	Operation string            `json:"operation"`
	Key       string            `json:"key"`
	RepoID    string            `json:"repoId,omitempty"`
	Target    Ref               `json:"target,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	At        time.Time         `json:"at"`
}

// AuditLog is an append-only, ordered record of forge operations.
type AuditLog struct {
	entries []Operation
}

// NewAuditLog returns an empty audit log.
func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

// Record appends an operation, assigning its sequence number.
func (l *AuditLog) Record(op Operation) Operation {
	op.Seq = len(l.entries) + 1
	l.entries = append(l.entries, op)
	return op
}

// Entries returns a copy of the recorded operations in order.
func (l *AuditLog) Entries() []Operation {
	out := make([]Operation, len(l.entries))
	copy(out, l.entries)
	return out
}

// Len returns the number of recorded operations.
func (l *AuditLog) Len() int {
	return len(l.entries)
}
