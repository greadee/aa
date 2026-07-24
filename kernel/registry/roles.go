package registry

import "sort"

// KnownRoles is the set of durable roles recognized by the registry.
var KnownRoles = map[Role]bool{
	"Construction Manager":  true,
	"Architect":             true,
	"Surveyor":              true,
	"Archivist":             true,
	"Planner":               true,
	"Foreman":               true,
	"Draftsman":             true,
	"Builder":               true,
	"General Contractor":    true,
	"Inspector":             true,
	"Safety Inspector":      true,
	"Structural Engineer":   true,
	"Adversarial Inspector": true,
	"Site Engineer":         true,
	"Commissioner":          true,
}

// capabilityRole maps a primary capability to the role that owns it.
var capabilityRole = map[string]Role{
	"read_project":         "Surveyor",
	"write_workspace":      "Builder",
	"execute_command":      "Builder",
	"run_tests":            "Inspector",
	"network_access":       "Site Engineer",
	"install_dependencies": "Site Engineer",
	"call_model":           "Builder",
	"read_secrets":         "Commissioner",
	"create_artifact":      "Builder",
	"open_pull_request":    "General Contractor",
	"merge":                "Commissioner",
	"deploy":               "Commissioner",
}

// RolesFor returns the deterministic roles implied by required capabilities.
func RolesFor(capabilities []string) []Role {
	seen := make(map[Role]bool)
	var out []Role
	for _, c := range capabilities {
		role, ok := capabilityRole[c]
		if !ok || seen[role] {
			continue
		}
		seen[role] = true
		out = append(out, role)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// IsKnownRole reports whether role is recognized.
func IsKnownRole(role Role) bool { return KnownRoles[role] }
