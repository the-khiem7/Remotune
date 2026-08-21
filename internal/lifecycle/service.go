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
	wailsapplication "github.com/wailsapp/wails/v3/pkg/application"
)

// Service owns the Remotune coordinator and CRD detector lifecycle. It bridges
// the Wails application events to the coordinator's methods.
type Service struct {
	mu           sync.Mutex
	coordinator  *application.Coordinator
	store        *application.RecoveryStore
	running      bool
	shuttingDown bool
	cancel       context.CancelFunc
	done         chan struct{}
	diagnostics  *application.DetectorDiagnostics
}

// NewService creates the lifecycle service with default configuration.
func NewService() *Service {
	return &Service{
		diagnostics: application.NewDetectorDiagnostics(),
	}
}

// ServiceStartup starts the coordinator loop after Wails has created the application
// context. Wails excludes this lifecycle method from the generated frontend bindings.
func (s *Service) ServiceStartup(ctx context.Context, _ wailsapplication.ServiceOptions) error {
	go s.run(ctx)
	return nil
}

// ServiceShutdown stops the detector and restores any owned Windows state during
// Wails shutdown. Wails excludes this lifecycle method from generated bindings.
func (s *Service) ServiceShutdown() error {
	return s.shutdown()
}

// Shutdown performs the native tray quit sequence without exposing a service method
// to the frontend binding surface.
func Shutdown(s *Service) error {
	if s == nil {
		return fmt.Errorf("lifecycle service is nil")
	}
	return s.shutdown()
}

// run starts the coordinator loop. It blocks until ctx is canceled or the app quits.
func (s *Service) run(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	if s.shuttingDown {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan struct{})

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		close(s.done)
		s.running = false
		s.mu.Unlock()
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
	if err := application.Run(runCtx, coord, det, s.diagnostics); err != nil {
		slog.Error("coordinator run loop exited with error", "error", err)
	}
}

// shutdown stops transitions, restores owned state, and cleans up resources.
func (s *Service) shutdown() error {
	s.mu.Lock()
	if s.shuttingDown {
		s.mu.Unlock()
		return nil
	}
	s.shuttingDown = true
	coord := s.coordinator
	cancelFn := s.cancel
	running := s.running
	done := s.done
	s.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}

	// Wait for Run to finish so no concurrent transitions are in flight. If Quit
	// was selected before ApplicationStarted, there is no loop or owned snapshot to
	// wait for.
	if running {
		<-done
	}

	if coord != nil {
		if err := coord.Quit(); err != nil {
			return fmt.Errorf("restore before quit: %w", err)
		}
	}
	return nil
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
			Detector:          s.diagnostics.Status(),
		}
	}
	status := coord.Status()
	status.Detector = s.diagnostics.Status()
	return status
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
