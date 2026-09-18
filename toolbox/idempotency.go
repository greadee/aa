package toolbox

import "strings"

// Key derives a stable idempotency key from an operation name and its parts.
//
// The host and the RPC adapter use the key to make invocation safe to repeat:
// the same key returns the original result instead of executing twice.
func Key(operation string, parts ...string) string {
	cleaned := make([]string, 0, len(parts)+1)
	cleaned = append(cleaned, strings.TrimSpace(operation))
	for _, part := range parts {
		cleaned = append(cleaned, strings.TrimSpace(part))
	}
	return strings.Join(cleaned, ":")
}

// InvocationKey derives the idempotency key for invoking a tool under an
// execution contract. An explicit key is namespaced by tool so two tools may
// reuse the same caller-supplied value without colliding.
func InvocationKey(toolID ToolID, contractID, explicit string) string {
	token := explicit
	if token == "" {
		token = contractID
	}
	return Key("invoke", string(toolID), token)
}
