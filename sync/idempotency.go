package sync

import "strings"

// Key derives a stable idempotency key from an operation name and its parts.
//
// Implementations use the key to make transfer and distribution safe to
// repeat: the same key returns the original receipt instead of moving the
// payload a second time.
func Key(operation string, parts ...string) string {
	cleaned := make([]string, 0, len(parts)+1)
	cleaned = append(cleaned, strings.TrimSpace(operation))
	for _, part := range parts {
		cleaned = append(cleaned, strings.TrimSpace(part))
	}
	return strings.Join(cleaned, ":")
}

// TransferKey derives the idempotency key for distributing a work package or
// fetching an artifact to a node.
func TransferKey(kind, objectID string, node NodeID) string {
	return Key(kind, objectID, string(node))
}
