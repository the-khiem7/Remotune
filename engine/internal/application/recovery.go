//go:build windows

// Package application implements the Phase 3 StateCoordinator: the single owner of
// all Windows state transitions, and the durable recovery store that survives a
// crash or restart while a temporary override is active.
//
// Per docs/baseline/remotune.sourcecode.md ("StateCoordinator"): detector callbacks,
// UI handlers, tray callbacks, startup, and recovery code submit observations and
// commands here; they never call wintune managers directly.
package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/khiemnguyen/remotune/engine/internal/wintune"
)

// recoveryFileName is the durable snapshot file. A single file, not per-category,
// because ownership is a single unit: a duplicate connect must not be able to
// replace one category's baseline while leaving another's stale (ledger decision 14).
const recoveryFileName = "recovery.json"

// ErrNoRecovery is returned when no valid, owned recovery snapshot exists. Callers
// must treat this as "nothing to restore," never as license to guess a baseline
// (docs/baseline/remotune.sourcecode.md, "Persistence" required properties).
var ErrNoRecovery = errors.New("no valid owned recovery snapshot")

// RecoveryStore persists the single active recovery snapshot durably, so a crash
// between apply and restore does not strand Windows in a temporary override with no
// way back to the user's real baseline.
type RecoveryStore struct {
	dir string
}

// NewRecoveryStore opens (without yet touching) the store at dir. If dir is empty,
// the default is %LOCALAPPDATA%\Remotune, per the persistence candidate location in
// the source architecture baseline.
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

// Save writes snap durably and atomically: a crash mid-write must never leave a
// corrupt or half-written recovery file, since that file is the only path back to
// the user's real baseline (docs/baseline/remotune.sourcecode.md, "written durably
// before apply begins").
//
// Atomicity is achieved by writing to a temp file in the same directory (so the
// rename is same-volume) and renaming over the target, which is atomic on NTFS.
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
	// If anything below fails, the temp file must not linger.
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

// Load reads the current recovery snapshot. It returns ErrNoRecovery, wrapped, if no
// file exists or the file fails validation — a snapshot that cannot be trusted is
// treated identically to no snapshot at all, never guessed around.
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

// Retire deletes the recovery snapshot. Callers must only do this after a verified
// successful restore (docs/baseline/remotune.sourcecode.md, "retired only after
// verified successful restoration"); Retire itself does not verify anything, it is a
// pure deletion, so the calling coordinator carries that responsibility.
func (s *RecoveryStore) Retire() error {
	err := os.Remove(s.path())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("retire recovery file: %w", err)
	}
	return nil
}

// Exists reports whether a recovery file is present, without validating it. Used for
// diagnostics; ownership decisions must go through Load, which validates.
func (s *RecoveryStore) Exists() bool {
	_, err := os.Stat(s.path())
	return err == nil
}
