package sandbox

// KindProcess is the isolation kind this sandbox enforces: a run confined to a
// working directory under a root, with an allowlisted environment and bounded
// time and output.
const KindProcess = "process"

// Enforcer reports the isolation kinds this sandbox can enforce. The kernel
// wires it into the toolbox policy engine's SandboxEnforcer seam, so a tool that
// requires isolation is allowed only when the runtime sandbox can provide it.
type Enforcer struct{}

// NewEnforcer returns an Enforcer.
func NewEnforcer() Enforcer { return Enforcer{} }

// CanEnforce reports whether kind is the process isolation kind.
func (Enforcer) CanEnforce(kind string) bool { return kind == KindProcess }
