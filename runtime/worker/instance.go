package worker

import "github.com/greadee/aa/registry/roles"

// WorkerID identifies a worker instance.
type WorkerID string

// Worker is a runtime instance: an instantiation of a role on a provider. It is
// ephemeral per assignment and is never a durable definition (those live in
// registry).
type Worker struct {
	ID           WorkerID
	Roles        []roles.Role
	Capabilities []string
	Available    bool
	Node         string
	Provider     string
	CostWeight   int
}
