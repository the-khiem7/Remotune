//go:build windows

package application

import (
	"testing"

	"github.com/khiemnguyen/remotune/internal/crd"
	"github.com/khiemnguyen/remotune/internal/wintune"
)

func TestProfileStoreRoundTripAndNormalizesCustomSelection(t *testing.T) {
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewProfileStore: %v", err)
	}
	settings := DefaultProfileSettings()
	settings.CRDOnProfile = wintune.ProfileCustom
	settings.CustomEffects = map[string]bool{"MenuAnimation": true}
	if err := store.Save(settings); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.CustomEffects["MenuAnimation"] {
		t.Fatal("saved custom selection was lost")
	}
	if len(got.CustomEffects) != len(wintune.EffectNames()) {
		t.Fatalf("custom effects = %d, want %d", len(got.CustomEffects), len(wintune.EffectNames()))
	}
}

func TestCustomSelectionRemainsCustomUntilAnEffectIsEdited(t *testing.T) {
	settings := DefaultProfileSettings()
	settings.CRDOnProfile = wintune.ProfileCustom
	settings.CustomEffects = allEffects(true)
	if got := settings.Normalized().CRDOnProfile; got != wintune.ProfileCustom {
		t.Fatalf("custom profile normalized to %q, want custom", got)
	}
}

func TestCRDOffProfileRetiresSnapshotAfterApplyingSelectedProfile(t *testing.T) {
	c, ve, tb, store := newTestCoordinator(t, fullCfg())
	settings := DefaultProfileSettings()
	settings.CRDOffAction = CRDOffBestAppearance
	if err := c.UpdateProfiles(settings); err != nil {
		t.Fatalf("UpdateProfiles: %v", err)
	}
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := c.Observe(crd.StateDisconnected); err != nil {
		t.Fatalf("Observe disconnected: %v", err)
	}
	if store.Exists() {
		t.Fatal("owned snapshot was not retired after verified selected off profile")
	}
	if ve.restoreCalls != 0 {
		t.Fatalf("visual effects restore calls = %d, want 0 for a selected off profile", ve.restoreCalls)
	}
	if tb.restoreCalls != 1 {
		t.Fatalf("taskbar restore calls = %d, want 1", tb.restoreCalls)
	}
}
