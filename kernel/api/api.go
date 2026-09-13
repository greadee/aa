// Package api exposes the kernel control plane as an in-process service.
//
// It is a thin, concurrency-safe surface over the orchestrator. Execution is
// disabled unless the service is explicitly enabled; a disabled service
// refuses to dispatch. The transport (HTTP/socket) lands with the console.
package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/greadee/aa/kernel/orchestrator"
)

// ErrDisabled is returned when the service is not enabled.
var ErrDisabled = errors.New("api: execution is disabled")

// Config configures the control-plane service.
type Config struct {
	Orchestrator *orchestrator.Orchestrator
	Enabled      bool
}

// Service is the control-plane surface.
type Service struct {
	mu      sync.Mutex
	orch    *orchestrator.Orchestrator
	enabled bool
}

// New returns a control-plane service.
func New(cfg Config) (*Service, error) {
	if cfg.Orchestrator == nil {
		return nil, fmt.Errorf("api: orchestrator is required")
	}
	return &Service{orch: cfg.Orchestrator, enabled: cfg.Enabled}, nil
}

// Enabled reports whether execution is enabled.
func (s *Service) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}

// Status returns the current project status.
func (s *Service) Status() orchestrator.ProjectStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orch.Status()
}

// Dispatch dispatches the next ready work package.
func (s *Service) Dispatch(ctx context.Context) (orchestrator.DispatchReport, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return orchestrator.DispatchReport{}, false, ErrDisabled
	}
	return s.orch.Dispatch(ctx)
}

// Approve records human approval for a work package.
func (s *Service) Approve(workPackageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orch.Approve(workPackageID)
}

// Resolve re-evaluates gates for a work package that is awaiting gates.
func (s *Service) Resolve(workPackageID string) (orchestrator.DispatchReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orch.Resolve(workPackageID)
}
