//go:build windows

package application

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/khiemnguyen/remotune/internal/wintune"
)

func validSnapshot() *wintune.Snapshot {
	return &wintune.Snapshot{
		SchemaVersion: wintune.SnapshotSchemaVersion,
		CapturedAt:    time.Now(),
		Machine:       "test-machine",
		VisualEffects: &wintune.VisualEffectsSnapshot{
			SPI:      map[string]int32{"MenuAnimation": 1},
			Registry: map[string]wintune.RegValue{"VisualFXSetting": {Kind: wintune.RegKindDWord, DWord: 1}},
			Mask:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		Taskbar: &wintune.TaskbarSnapshot{AutoHide: true, ABMState: 1, LivePersistedAgreed: true},
	}
}

func newTestStore(t *testing.T) *RecoveryStore {
	t.Helper()
	dir := t.TempDir()
	s, err := NewRecoveryStore(dir)
	if err != nil {
		t.Fatalf("NewRecoveryStore: %v", err)
	}
	return s
}

func TestRecoveryStoreLoadWithNoFileIsErrNoRecovery(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Load()
	if !errors.Is(err, ErrNoRecovery) {
		t.Fatalf("Load() error = %v, want ErrNoRecovery", err)
	}
}

func TestRecoveryStoreSaveLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := validSnapshot()
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !s.Exists() {
		t.Fatal("Exists() = false after Save")
	}

	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Machine != want.Machine {
		t.Errorf("Machine = %q, want %q", got.Machine, want.Machine)
	}
	if len(got.VisualEffects.SPI) != len(want.VisualEffects.SPI) {
		t.Errorf("VisualEffects.SPI length mismatch")
	}
}

func TestRecoveryStoreSaveRejectsInvalidSnapshot(t *testing.T) {
	s := newTestStore(t)
	bad := &wintune.Snapshot{SchemaVersion: wintune.SnapshotSchemaVersion} // no categories
	if err := s.Save(bad); err == nil {
		t.Fatal("Save should have rejected an invalid snapshot rather than persisting it")
	}
	if s.Exists() {
		t.Fatal("an invalid snapshot must not be written to disk at all")
	}
}

func TestRecoveryStoreLoadRejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	s, err := NewRecoveryStore(dir)
	if err != nil {
		t.Fatalf("NewRecoveryStore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, recoveryFileName), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	_, err = s.Load()
	if !errors.Is(err, ErrNoRecovery) {
		t.Fatalf("Load() on corrupt file error = %v, want wrapping ErrNoRecovery", err)
	}
}

func TestRecoveryStoreLoadRejectsWrongSchemaVersion(t *testing.T) {
	s := newTestStore(t)
	snap := validSnapshot()
	// Bypass Save's validation to write a future-schema file directly, simulating an
	// old Remotune reading a snapshot from a newer one (or vice versa).
	snap.SchemaVersion = wintune.SnapshotSchemaVersion + 1
	data, _ := json.Marshal(snap)
	if err := os.WriteFile(filepath.Join(s.dir, recoveryFileName), data, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := s.Load()
	if !errors.Is(err, ErrNoRecovery) {
		t.Fatalf("Load() with mismatched schema version error = %v, want wrapping ErrNoRecovery", err)
	}
}

func TestRecoveryStoreRetireIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	if err := s.Retire(); err != nil {
		t.Fatalf("Retire on empty store: %v", err)
	}
	if err := s.Save(validSnapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Retire(); err != nil {
		t.Fatalf("Retire: %v", err)
	}
	if s.Exists() {
		t.Fatal("Exists() = true after Retire")
	}
	if err := s.Retire(); err != nil {
		t.Fatalf("second Retire should be a no-op, got: %v", err)
	}
}

func TestRecoveryStoreSaveOverwritesAtomically(t *testing.T) {
	s := newTestStore(t)
	first := validSnapshot()
	first.Machine = "first"
	if err := s.Save(first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	second := validSnapshot()
	second.Machine = "second"
	if err := s.Save(second); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Machine != "second" {
		t.Errorf("Machine = %q, want %q (overwrite must fully replace, no stale merge)", got.Machine, "second")
	}
	// No leftover temp files after a successful save.
	entries, _ := os.ReadDir(s.dir)
	for _, e := range entries {
		if e.Name() != recoveryFileName {
			t.Errorf("unexpected leftover file in recovery dir: %s", e.Name())
		}
	}
}
