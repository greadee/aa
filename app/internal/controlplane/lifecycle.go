package controlplane

import "net/http"

// WorkerLifecycle is a read view of one runtime worker instance.
type WorkerLifecycle struct {
	ID             string   `json:"id"`
	SelectedWorker string   `json:"selectedWorker,omitempty"`
	AssignmentID   string   `json:"assignmentId,omitempty"`
	WorkPackageID  string   `json:"workPackageId,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	State          string   `json:"state"`
	CreatedAt      string   `json:"createdAt,omitempty"`
	EndedAt        string   `json:"endedAt,omitempty"`
	Status         string   `json:"status,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Error          string   `json:"error,omitempty"`
}

// WorkerLifecycleEvent is one lifecycle transition.
type WorkerLifecycleEvent struct {
	WorkerID      string `json:"workerId"`
	AssignmentID  string `json:"assignmentId,omitempty"`
	WorkPackageID string `json:"workPackageId,omitempty"`
	State         string `json:"state"`
	Reason        string `json:"reason,omitempty"`
	At            string `json:"at,omitempty"`
}

// LifecycleBackend serves the runtime worker lifecycle read model. It is
// implemented by the composition root; the transport never imports the runtime
// module.
type LifecycleBackend interface {
	Workers() []WorkerLifecycle
	WorkerEvents() []WorkerLifecycleEvent
}

func (s *Server) workers(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Lifecycle == nil {
		writeJSON(w, http.StatusOK, []WorkerLifecycle{})
		return
	}
	workers := s.cfg.Lifecycle.Workers()
	if workers == nil {
		workers = []WorkerLifecycle{}
	}
	writeJSON(w, http.StatusOK, workers)
}

func (s *Server) workerEvents(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.Lifecycle == nil {
		writeJSON(w, http.StatusOK, []WorkerLifecycleEvent{})
		return
	}
	events := s.cfg.Lifecycle.WorkerEvents()
	if events == nil {
		events = []WorkerLifecycleEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}
