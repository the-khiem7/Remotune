//go:build windows

// Package lifecycle wires the Remotune coordinator to the Wails application lifecycle:
// startup, tray, autostart, quit, and WebView2 prerequisite detection.
//
// It does NOT import the wintune or crd packages directly at the Service layer —
// it operates through the coordinator, preserving the Phase 3 invariant that only
// the coordinator calls adapters.
package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/khiemnguyen/remotune/internal/application"
	"github.com/khiemnguyen/remotune/internal/crd"
	"github.com/khiemnguyen/remotune/internal/wintune"
)

// Service owns the Remotune coordinator and CRD detector lifecycle. It bridges
// the Wails application events to the coordinator's methods.
type Service struct {
	mu          sync.Mutex
	coordinator *application.Coordinator
	store       *application.RecoveryStore
	running     bool
	cancel      context.CancelFunc
	done        chan struct{}
}

// NewService creates the lifecycle service with default configuration. The actual
// coordinator start happens in Run, which is called after the Wails app is running.
func NewService() *Service {
	return &Service{
		done: make(chan struct{}),
	}
}

// Run starts the coordinator loop. It blocks until ctx is canceled or Quit is called.
// This must be called from a goroutine after Wails ApplicationStarted.
func (s *Service) Run(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		close(s.done)
	}()

	// Initialize adapters.
	ve := &wintune.VisualEffectsManager{}
	tb := &wintune.TaskbarManager{}

	store, err := application.NewRecoveryStore("")
	if err != nil {
		slog.Error("failed to initialize recovery store", "error", err)
		return
	}
	s.mu.Lock()
	s.store = store
	s.mu.Unlock()

	cfg := application.AutomationConfig{
		Enabled:       true,
		VisualEffects: true,
		Taskbar:       true,
	}

	coord := application.NewCoordinator(store, ve, tb, cfg)
	s.mu.Lock()
	s.coordinator = coord
	s.mu.Unlock()

	det := application.LiveDetector{}
	if err := application.Run(runCtx, coord, det); err != nil {
		slog.Error("coordinator run loop exited with error", "error", err)
	}
}

// Shutdown performs the explicit Quit sequence: stop transitions, restore owned state,
// and clean up resources. Called by the Wails shutdown hook.
func (s *Service) Shutdown() {
	s.mu.Lock()
	coord := s.coordinator
	cancelFn := s.cancel
	s.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}

	// Wait for Run to finish so no concurrent transitions are in flight.
	<-s.done

	if coord != nil {
		if err := coord.Quit(); err != nil {
			slog.Error("coordinator quit returned error", "error", err)
		}
	}
}

// Status returns the current coordinator status for UI display.
func (s *Service) Status() application.Status {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return application.Status{
			Tuning:            application.TuningUnknown,
			CRD:               crd.StateUnknown,
			AutomationEnabled: true,
		}
	}
	return coord.Status()
}

// Pause pauses automation, restoring owned state.
func (s *Service) Pause() error {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return coord.Pause()
}

// Resume resumes automation.
func (s *Service) Resume() error {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return coord.Resume()
}

// RestoreNow manually triggers a restore.
func (s *Service) RestoreNow() error {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return coord.RestoreNow()
}

// GetAutostartStatus returns the authoritative Windows startup registration
// state. It is exposed to the frontend so the UI never infers the result of a
// previous toggle.
func (s *Service) GetAutostartStatus() (AutostartStatus, error) {
	return GetAutostartStatus()
}

// SetAutostart changes the Windows startup registration and returns the state
// read back from the registry after the operation.
func (s *Service) SetAutostart(enabled bool) (AutostartStatus, error) {
	if err := SetAutostart(enabled); err != nil {
		return AutostartStatus{}, err
	}
	return GetAutostartStatus()
}

// PortablePathStatus exposes an actionable diagnostic when the executable has
// been moved after Start with Windows was enabled.
func (s *Service) PortablePathStatus() PortablePathStatus {
	return CheckPortablePath()
}
