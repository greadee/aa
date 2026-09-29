// Package capabilities holds aa's closed execution-capability vocabulary.
//
// A capability is a durable definition: an authority a work package may be
// granted under an execution contract. Whether a specific attempt is granted
// or denied a capability is decided at runtime by the kernel, not here.
package capabilities

// Known is the closed capability vocabulary.
var Known = map[string]bool{
	"read_project":         true,
	"write_workspace":      true,
	"execute_command":      true,
	"run_tests":            true,
	"network_access":       true,
	"install_dependencies": true,
	"call_model":           true,
	"read_secrets":         true,
	"create_artifact":      true,
	"open_pull_request":    true,
	"merge":                true,
	"deploy":               true,
}
