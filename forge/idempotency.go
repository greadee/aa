package forge

import "strings"

// Key derives a stable idempotency key from an operation name and its parts.
//
// Implementations use the key to make mutating operations safe to repeat: the
// same key returns the original result instead of creating a duplicate object.
func Key(operation string, parts ...string) string {
	cleaned := make([]string, 0, len(parts)+1)
	cleaned = append(cleaned, strings.TrimSpace(operation))
	for _, part := range parts {
		cleaned = append(cleaned, strings.TrimSpace(part))
	}
	return strings.Join(cleaned, ":")
}
