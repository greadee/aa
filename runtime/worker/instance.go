package worker

import "github.com/greadee/aa/registry/roles"

// Trade is a durable capability (for example "backend"). It is a transitional
// field; the target ontology is Role × Model with the capability tier supplied
// by the allocator.
type Trade string

// WorkerID identifies a worker instance.
type WorkerID string

// Worker is a runtime instance: an instantiation of a role on a provider. It is
// ephemeral per assignment and is never a durable definition (those live in
// registry).
type Worker struct {
	ID           WorkerID
	Trade        Trade
	Roles        []roles.Role
	Capabilities []string
	Available    bool
	Node         string
	Provider     string
	CostWeight   int
}
