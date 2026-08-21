//go:build windows

package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/khiemnguyen/remotune/internal/wintune"
)

const recoveryFileName = "recovery.json"

var ErrNoRecovery = errors.New("no valid owned recovery snapshot")

type RecoveryStore struct {
	dir string
}

func NewRecoveryStore(dir string) (*RecoveryStore, error) {
	if dir == "" {
		base, err := os.UserCacheDir() // %LOCALAPPDATA% on Windows
		if err != nil {
			return nil, fmt.Errorf("resolve LOCALAPPDATA: %w", err)
		}
		dir = filepath.Join(base, "Remotune")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create recovery dir: %w", err)
	}
	return &RecoveryStore{dir: dir}, nil
}

func (s *RecoveryStore) path() string {
	return filepath.Join(s.dir, recoveryFileName)
}
func (s *RecoveryStore) Save(snap *wintune.Snapshot) error {
	if err := snap.Validate(); err != nil {
		return fmt.Errorf("refusing to save an invalid snapshot: %w", err)
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	tmp, err := os.CreateTemp(s.dir, "recovery-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp recovery file: %w", err)
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp recovery file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp recovery file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp recovery file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path()); err != nil {
		return fmt.Errorf("commit recovery file: %w", err)
	}
	success = true
	return nil
}
func (s *RecoveryStore) Load() (*wintune.Snapshot, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoRecovery
		}
		return nil, fmt.Errorf("read recovery file: %w", err)
	}

	var snap wintune.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("%w: corrupt recovery file: %v", ErrNoRecovery, err)
	}
	if err := snap.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoRecovery, err)
	}
	return &snap, nil
}
func (s *RecoveryStore) Retire() error {
	err := os.Remove(s.path())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("retire recovery file: %w", err)
	}
	return nil
}
func (s *RecoveryStore) Exists() bool {
	_, err := os.Stat(s.path())
	return err == nil
}
