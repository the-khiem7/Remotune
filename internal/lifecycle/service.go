//go:build windows

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

type Service struct {
	mu           sync.Mutex
	coordinator  *application.Coordinator
	store        *application.RecoveryStore
	profiles     *application.ProfileStore
	running      bool
	shuttingDown bool
	cancel       context.CancelFunc
	done         chan struct{}
	diagnostics  *application.DetectorDiagnostics
}

func NewService() *Service {
	return &Service{
		diagnostics: application.NewDetectorDiagnostics(),
	}
}
func (s *Service) ServiceStartup(ctx context.Context, _ wailsapplication.ServiceOptions) error {
	go s.run(ctx)
	return nil
}
func (s *Service) ServiceShutdown() error {
	return s.shutdown()
}
func Shutdown(s *Service) error {
	if s == nil {
		return fmt.Errorf("lifecycle service is nil")
	}
	return s.shutdown()
}
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
	profileStore, err := application.NewProfileStore("")
	if err != nil {
		slog.Error("failed to initialize profile store", "error", err)
		return
	}
	profileSettings, err := profileStore.Load()
	if err != nil {
		slog.Error("failed to load profile settings", "error", err)
		return
	}
	s.mu.Lock()
	s.profiles = profileStore
	s.mu.Unlock()

	cfg := application.AutomationConfig{
		Enabled:       true,
		VisualEffects: true,
		Taskbar:       true,
		Profiles:      profileSettings,
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
func (s *Service) Pause() error {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return coord.Pause()
}
func (s *Service) Resume() error {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return coord.Resume()
}
func (s *Service) RestoreNow() error {
	s.mu.Lock()
	coord := s.coordinator
	s.mu.Unlock()

	if coord == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return coord.RestoreNow()
}
func (s *Service) GetAutostartStatus() (AutostartStatus, error) {
	return GetAutostartStatus()
}
func (s *Service) SetAutostart(enabled bool) (AutostartStatus, error) {
	if err := SetAutostart(enabled); err != nil {
		return AutostartStatus{}, err
	}
	return GetAutostartStatus()
}
func (s *Service) PortablePathStatus() PortablePathStatus {
	return CheckPortablePath()
}
func (s *Service) GetProfileSettings() (application.ProfileSettings, error) {
	s.mu.Lock()
	store := s.profiles
	s.mu.Unlock()
	if store == nil {
		return application.DefaultProfileSettings(), nil
	}
	return store.Load()
}
func (s *Service) SetProfileSettings(settings application.ProfileSettings) (application.ProfileSettings, error) {
	settings = settings.Normalized()
	if err := settings.Validate(); err != nil {
		return application.ProfileSettings{}, err
	}
	s.mu.Lock()
	store := s.profiles
	coord := s.coordinator
	s.mu.Unlock()
	if store == nil || coord == nil {
		return application.ProfileSettings{}, fmt.Errorf("profile settings are not initialized")
	}
	if err := store.Save(settings); err != nil {
		return application.ProfileSettings{}, err
	}
	if err := coord.UpdateProfiles(settings); err != nil {
		return application.ProfileSettings{}, err
	}
	return settings, nil
}
func (s *Service) VisualEffectNames() []string {
	return wintune.EffectNames()
}
