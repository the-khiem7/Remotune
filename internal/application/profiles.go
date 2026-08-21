//go:build windows

package application

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/khiemnguyen/remotune/internal/wintune"
)

const profileSettingsSchemaVersion = 1

type CRDOffAction string

const (
	CRDOffRestoreSnapshot CRDOffAction = "restoreSnapshot"
	CRDOffWindowsChoose   CRDOffAction = "windowsChoose"
	CRDOffBestAppearance  CRDOffAction = "bestAppearance"
	CRDOffBestPerformance CRDOffAction = "bestPerformance"
)

type ProfileSettings struct {
	SchemaVersion int                          `json:"schemaVersion"`
	CRDOnProfile  wintune.VisualEffectsProfile `json:"crdOnProfile"`
	CRDOffAction  CRDOffAction                 `json:"crdOffAction"`
	CustomEffects map[string]bool              `json:"customEffects"`
}

func DefaultProfileSettings() ProfileSettings {
	return ProfileSettings{
		SchemaVersion: profileSettingsSchemaVersion,
		CRDOnProfile:  wintune.ProfileBestPerformance,
		CRDOffAction:  CRDOffRestoreSnapshot,
		CustomEffects: allEffects(true),
	}
}

func (s ProfileSettings) Validate() error {
	if s.SchemaVersion != profileSettingsSchemaVersion {
		return fmt.Errorf("unsupported profile settings schema %d", s.SchemaVersion)
	}
	if !s.CRDOnProfile.Valid() {
		return fmt.Errorf("unsupported CRD-on profile %q", s.CRDOnProfile)
	}
	switch s.CRDOffAction {
	case CRDOffRestoreSnapshot, CRDOffWindowsChoose, CRDOffBestAppearance, CRDOffBestPerformance:
	default:
		return fmt.Errorf("unsupported CRD-off action %q", s.CRDOffAction)
	}
	for name := range s.CustomEffects {
		if !isEffectName(name) {
			return fmt.Errorf("unsupported custom effect %q", name)
		}
	}
	return nil
}

func (s ProfileSettings) Normalized() ProfileSettings {
	if s.SchemaVersion == 0 {
		s = DefaultProfileSettings()
	}
	if s.CustomEffects == nil {
		s.CustomEffects = map[string]bool{}
	}
	for _, name := range wintune.EffectNames() {
		if _, ok := s.CustomEffects[name]; !ok {
			s.CustomEffects[name] = false
		}
	}
	return s
}

func (s ProfileSettings) OffProfile() wintune.VisualEffectsProfile {
	switch s.CRDOffAction {
	case CRDOffWindowsChoose:
		return wintune.ProfileWindowsChoose
	case CRDOffBestAppearance:
		return wintune.ProfileBestAppearance
	default:
		return wintune.ProfileBestPerformance
	}
}

type ProfileStore struct{ path string }

func NewProfileStore(dir string) (*ProfileStore, error) {
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("resolve LOCALAPPDATA: %w", err)
		}
		dir = filepath.Join(base, "Remotune")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create profile settings dir: %w", err)
	}
	return &ProfileStore{path: filepath.Join(dir, "profiles.json")}, nil
}

func (s *ProfileStore) Load() (ProfileSettings, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return DefaultProfileSettings(), nil
	}
	if err != nil {
		return ProfileSettings{}, fmt.Errorf("read profile settings: %w", err)
	}
	var settings ProfileSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return ProfileSettings{}, fmt.Errorf("decode profile settings: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return ProfileSettings{}, err
	}
	return settings.Normalized(), nil
}

func (s *ProfileStore) Save(settings ProfileSettings) error {
	settings = settings.Normalized()
	if err := settings.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile settings: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "profiles-*.tmp")
	if err != nil {
		return fmt.Errorf("create profile settings temp file: %w", err)
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write profile settings: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync profile settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close profile settings: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("commit profile settings: %w", err)
	}
	success = true
	return nil
}

func allEffects(value bool) map[string]bool {
	effects := map[string]bool{}
	for _, name := range wintune.EffectNames() {
		effects[name] = value
	}
	return effects
}

func isEffectName(name string) bool {
	for _, candidate := range wintune.EffectNames() {
		if candidate == name {
			return true
		}
	}
	return false
}
