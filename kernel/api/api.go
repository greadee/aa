// Package api exposes the kernel control plane as an in-process service.
//
// It is a thin, concurrency-safe surface over the scheduler. Execution is
// disabled unless the service is explicitly enabled; a disabled service
// refuses to dispatch. The transport (HTTP/socket) lands with the console.
package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/greadee/aa/kernel/scheduler"
)

// ErrDisabled is returned when the service is not enabled.
var ErrDisabled = errors.New("api: execution is disabled")

// Config configures the control-plane service.
type Config struct {
	Scheduler *scheduler.Scheduler
	Enabled   bool
}

// Service is the control-plane surface.
type Service struct {
	mu      sync.Mutex
	orch    *scheduler.Scheduler
	enabled bool
}

// New returns a control-plane service.
func New(cfg Config) (*Service, error) {
	if cfg.Scheduler == nil {
		return nil, fmt.Errorf("api: scheduler is required")
	}
	return &Service{orch: cfg.Scheduler, enabled: cfg.Enabled}, nil
}

// Enabled reports whether execution is enabled.
func (s *Service) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}

// Status returns the current project status.
func (s *Service) Status() scheduler.ProjectStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orch.Status()
}

// Dispatch dispatches the next ready work package.
func (s *Service) Dispatch(ctx context.Context) (scheduler.DispatchReport, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return scheduler.DispatchReport{}, false, ErrDisabled
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
func (s *Service) Resolve(workPackageID string) (scheduler.DispatchReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orch.Resolve(workPackageID)
}
