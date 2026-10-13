package controlplane

// Error codes aligned with the RPC vocabulary in contracts/rpc/rpc-v1.md. The
// control plane reuses the same governed `aa.*` codes so callers see one error
// language across boundaries.
const (
	codeInvalidParams  = "aa.invalid_params"
	codeNotFound       = "aa.not_found"
	codeMethodNotFound = "aa.method_not_found"
	codeUnauthorized   = "aa.unauthorized"
	codeUnavailable    = "aa.unavailable"
	codeConflict       = "aa.conflict"
)
