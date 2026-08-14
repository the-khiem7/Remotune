//go:build windows

package wintune

import (
	"errors"
	"os"
	"testing"
	"time"
)

// Mutating tests reconfigure the live desktop, so they are opt-in. Set
// REMOTUNE_SYSTEM_TESTS=1 to run them. They always restore the original state.
const systemTestEnv = "REMOTUNE_SYSTEM_TESTS"

func requireSystemTests(t *testing.T) {
	t.Helper()
	if os.Getenv(systemTestEnv) != "1" {
		t.Skipf("skipping system-mutating test; set %s=1 to run", systemTestEnv)
	}
}

// ---------- pure tests ----------

func TestSnapshotValidate(t *testing.T) {
	good := &VisualEffectsSnapshot{
		SPI:      map[string]int32{"MenuAnimation": 1},
		Registry: map[string]RegValue{"VisualFXSetting": {Kind: RegKindDWord, DWord: 1}},
		Mask:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}

	cases := []struct {
		name    string
		snap    *Snapshot
		wantErr bool
	}{
		{"nil", nil, true},
		{"wrong schema", &Snapshot{SchemaVersion: 99, VisualEffects: good}, true},
		{"no categories", &Snapshot{SchemaVersion: SnapshotSchemaVersion}, true},
		{"missing mask", &Snapshot{SchemaVersion: SnapshotSchemaVersion, VisualEffects: &VisualEffectsSnapshot{
			SPI: good.SPI, Registry: good.Registry,
		}}, true},
		{"missing spi", &Snapshot{SchemaVersion: SnapshotSchemaVersion, VisualEffects: &VisualEffectsSnapshot{
			Registry: good.Registry, Mask: good.Mask,
		}}, true},
		{"taskbar only is valid", &Snapshot{SchemaVersion: SnapshotSchemaVersion, Taskbar: &TaskbarSnapshot{AutoHide: true}}, false},
		{"complete", &Snapshot{SchemaVersion: SnapshotSchemaVersion, VisualEffects: good}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.snap.Validate()
			if c.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestUnsupportedSchemaIsIdentifiable(t *testing.T) {
	s := &Snapshot{SchemaVersion: SnapshotSchemaVersion + 1, Taskbar: &TaskbarSnapshot{}}
	if err := s.Validate(); !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("want ErrUnsupportedSchema, got %v", err)
	}
}

func TestResultPartialAndVerified(t *testing.T) {
	boom := errors.New("boom")

	full := Result{Categories: []CategoryResult{
		{Category: CategoryVisualEffects, Verified: true},
		{Category: CategoryTaskbar, Verified: true},
	}}
	if !full.FullyVerified() || full.Failed() || full.Partial() {
		t.Fatal("all-verified result misclassified")
	}

	partial := Result{Categories: []CategoryResult{
		{Category: CategoryVisualEffects, Verified: true},
		{Category: CategoryTaskbar, Err: boom},
	}}
	if !partial.Partial() {
		t.Fatal("partial failure not detected")
	}
	if partial.FullyVerified() {
		t.Fatal("partial failure must never report full success")
	}
	if partial.Err() == nil {
		t.Fatal("partial failure must surface an error")
	}

	// A write that succeeded but was not confirmed is not full success either.
	unverified := Result{Categories: []CategoryResult{{Category: CategoryTaskbar, Changed: true}}}
	if unverified.FullyVerified() {
		t.Fatal("unverified write must not report full success")
	}
}

func TestRestoreRefusesMissingSnapshot(t *testing.T) {
	var ve VisualEffectsManager
	if _, err := ve.Restore(nil); err == nil {
		t.Fatal("visual effects restore must refuse a nil snapshot rather than guess")
	}
	if _, err := ve.Restore(&VisualEffectsSnapshot{}); err == nil {
		t.Fatal("visual effects restore must refuse an incomplete snapshot")
	}

	var tb TaskbarManager
	if _, err := tb.Restore(nil); err == nil {
		t.Fatal("taskbar restore must refuse a nil snapshot")
	}
}

func TestBestPerformanceLeavesUntouchedEffectsAlone(t *testing.T) {
	// Guards the correctness property that the preset does not disable everything.
	untouched := []string{
		"GradientCaptions", "KeyboardCues", "HotTracking",
		"MenuFade", "TooltipFade", "FlatMenu", "UIEffects",
	}
	for _, name := range untouched {
		for _, off := range bestPerformanceOff {
			if off == name {
				t.Fatalf("%s must not be in bestPerformanceOff; the Windows preset leaves it alone", name)
			}
		}
	}
	if len(bestPerformanceOff) != 10 {
		t.Fatalf("bestPerformanceOff has %d entries, want the 10 the preset changes", len(bestPerformanceOff))
	}
	// IconsOnly is the one value that rises.
	if bestPerformanceRegistry["Advanced.IconsOnly"] != 1 {
		t.Fatal("Advanced.IconsOnly must be raised to 1, not zeroed")
	}
}

func TestMaskClearBitsMatchRecordedEvidence(t *testing.T) {
	want := [8]byte{0x0E, 0x2C, 0x04, 0x00, 0x02, 0x00, 0x00, 0x00}
	if maskClearBits != want {
		t.Fatalf("maskClearBits = %v, want %v", maskClearBits, want)
	}
	// The two unexplained bits must never appear in the clear mask.
	if maskClearBits[2]&0x01 != 0 {
		t.Fatal("byte[2]:0x01 is unexplained and must be preserved")
	}
	if maskClearBits[4]&0x10 != 0 {
		t.Fatal("byte[4]:0x10 is unexplained and must be preserved")
	}
}

// ---------- read-only system tests ----------

func TestSnapshotCapturesAllThreeLayers(t *testing.T) {
	var m VisualEffectsManager
	s, err := m.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if len(s.SPI) != len(spiEffects)+1 {
		t.Fatalf("captured %d SPI values, want %d", len(s.SPI), len(spiEffects)+1)
	}
	if len(s.Mask) < 5 {
		t.Fatalf("mask is %d bytes, too short to be valid", len(s.Mask))
	}
	for _, required := range []string{"VisualFXSetting", "Desktop.FontSmoothing", "Advanced.IconsOnly"} {
		if _, ok := s.Registry[required]; !ok {
			t.Errorf("registry layer is missing %s", required)
		}
	}
}

func TestTaskbarReadsBothLayers(t *testing.T) {
	var m TaskbarManager
	s, err := m.GetState()
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if !s.Agreed() {
		t.Logf("WARNING: live=%v persisted=%v disagree; an override here could revert spontaneously",
			s.Live, s.Persisted)
	}
}

// ---------- mutating system tests ----------

// TestTaskbarRoundTripBothDirections covers the acceptance-gate case Phase 0 missed:
// a baseline of auto-hide OFF must stay OFF, not be forced ON.
func TestTaskbarRoundTripBothDirections(t *testing.T) {
	requireSystemTests(t)
	var m TaskbarManager

	original, err := m.Snapshot()
	if err != nil {
		t.Fatalf("baseline snapshot failed: %v", err)
	}
	t.Cleanup(func() {
		if _, err := m.Restore(original); err != nil {
			t.Errorf("FAILED TO RESTORE taskbar to %v: %v", original.AutoHide, err)
		}
	})

	for _, baseline := range []bool{true, false} {
		t.Run(map[bool]string{true: "baseline_ON", false: "baseline_OFF"}[baseline], func(t *testing.T) {
			if _, err := m.SetAutoHide(baseline); err != nil {
				t.Fatalf("could not establish baseline %v: %v", baseline, err)
			}
			base, err := m.Snapshot()
			if err != nil {
				t.Fatalf("snapshot failed: %v", err)
			}
			if base.AutoHide != baseline {
				t.Fatalf("baseline is %v, want %v", base.AutoHide, baseline)
			}

			// Remotune's override always disables auto-hide.
			if _, err := m.SetAutoHide(false); err != nil {
				t.Fatalf("apply failed: %v", err)
			}

			res, err := m.Restore(base)
			if err != nil {
				t.Fatalf("restore failed: %v", err)
			}
			if !res.Verified {
				t.Fatal("restore was not verified")
			}

			got, err := m.GetState()
			if err != nil {
				t.Fatalf("read-back failed: %v", err)
			}
			if got.Live != baseline {
				t.Errorf("live auto-hide is %v, want %v", got.Live, baseline)
			}
			if !got.Agreed() {
				t.Errorf("layers disagree after restore: live=%v persisted=%v", got.Live, got.Persisted)
			}
		})
	}
}

// TestVisualEffectsCustomRoundTrip is the Phase 1 acceptance gate: an arbitrary Custom
// configuration must survive an ApplyBestPerformance / Restore cycle exactly.
func TestVisualEffectsCustomRoundTrip(t *testing.T) {
	requireSystemTests(t)
	var m VisualEffectsManager

	original, err := m.Snapshot()
	if err != nil {
		t.Fatalf("baseline snapshot failed: %v", err)
	}
	t.Cleanup(func() {
		if _, err := m.Restore(original); err != nil {
			t.Errorf("FAILED TO RESTORE visual effects: %v", err)
		}
	})

	// Establish an arbitrary Custom state that matches neither preset by writing the
	// individual effects, then capture what Windows actually holds.
	//
	// The state is NOT built as a synthetic snapshot and handed to Restore: a hand-made
	// snapshot can describe per-effect values and a mask that contradict each other,
	// which is not a state Windows can ever be in and therefore not a fair test.
	wantCustom := map[string]int32{
		"MenuAnimation":          0,
		"ComboBoxAnimation":      1,
		"ListBoxSmoothScrolling": 0,
		"GradientCaptions":       0,
		"HotTracking":            0,
		"MenuFade":               0,
		"SelectionFade":          1,
		"TooltipAnimation":       0,
		"CursorShadow":           0,
		"DropShadow":             1,
		"ClientAreaAnimation":    0,
	}
	// Build a fully self-consistent target and converge to it through Restore.
	//
	// Writing the per-effect values directly while leaving the mask describing something
	// else made this test flaky, because the live session and the persisted mask
	// disagreed and a shell reload could undo the writes. Restore derives the mask from
	// the per-effect values, so the two layers agree by construction.
	customTarget := cloneVE(original)
	for name, v := range wantCustom {
		customTarget.SPI[name] = v
	}
	customTarget.SPI[minAnimateKey] = 0
	customTarget.Registry["VisualFXSetting"] = RegValue{Kind: RegKindDWord, DWord: 3}
	customTarget.Registry["Advanced.IconsOnly"] = RegValue{Kind: RegKindDWord, DWord: 1}
	customTarget.Registry["Advanced.ListviewShadow"] = RegValue{Kind: RegKindDWord, DWord: 0}

	if _, err := m.Restore(customTarget); err != nil {
		t.Fatalf("could not establish the custom baseline: %v", err)
	}

	// This captured state, not the wish list above, is the baseline to round-trip.
	baseline, err := m.Snapshot()
	if err != nil {
		t.Fatalf("custom baseline snapshot failed: %v", err)
	}
	for name, want := range wantCustom {
		if got := baseline.SPI[name]; got != want {
			t.Fatalf("custom baseline did not take: %s is %d, want %d", name, got, want)
		}
	}
	if v := baseline.Registry["VisualFXSetting"]; v.DWord != 3 {
		t.Fatalf("custom baseline VisualFXSetting is %s, want 3", v)
	}

	applyRes, err := m.ApplyBestPerformance()
	if err != nil {
		t.Fatalf("ApplyBestPerformance failed: %v", err)
	}
	if !applyRes.Verified {
		t.Fatal("apply was not verified")
	}

	// The two unexplained mask bits must survive the apply.
	afterApply, err := m.Snapshot()
	if err != nil {
		t.Fatalf("snapshot after apply failed: %v", err)
	}
	assertBitPreserved(t, "byte[2]:0x01", baseline.Mask, afterApply.Mask, 2, 0x01)
	assertBitPreserved(t, "byte[4]:0x10", baseline.Mask, afterApply.Mask, 4, 0x10)

	restoreRes, err := m.Restore(baseline)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !restoreRes.Verified {
		t.Fatal("restore was not verified")
	}

	final, err := m.Snapshot()
	if err != nil {
		t.Fatalf("final snapshot failed: %v", err)
	}
	if diff := DiffVisualEffects(baseline, final); len(diff) > 0 {
		t.Fatalf("custom round-trip was not exact, %d value(s) differ: %v", len(diff), diff)
	}
}

func TestApplyBestPerformanceIsIdempotent(t *testing.T) {
	requireSystemTests(t)
	var m VisualEffectsManager

	original, err := m.Snapshot()
	if err != nil {
		t.Fatalf("baseline snapshot failed: %v", err)
	}
	t.Cleanup(func() {
		if _, err := m.Restore(original); err != nil {
			t.Errorf("FAILED TO RESTORE visual effects: %v", err)
		}
	})

	if _, err := m.ApplyBestPerformance(); err != nil {
		t.Fatalf("first apply failed: %v", err)
	}
	first, err := m.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	res, err := m.ApplyBestPerformance()
	if err != nil {
		t.Fatalf("second apply failed: %v", err)
	}
	second, err := m.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if diff := DiffVisualEffects(first, second); len(diff) > 0 {
		t.Fatalf("apply is not idempotent, %d value(s) changed on the second pass: %v", len(diff), diff)
	}
	if res.Changed {
		t.Error("second apply reported a change but nothing differed")
	}
}

// ---------- helpers ----------

func cloneVE(s *VisualEffectsSnapshot) *VisualEffectsSnapshot {
	out := &VisualEffectsSnapshot{
		SPI:      make(map[string]int32, len(s.SPI)),
		Registry: make(map[string]RegValue, len(s.Registry)),
		Mask:     make([]byte, len(s.Mask)),
	}
	for k, v := range s.SPI {
		out.SPI[k] = v
	}
	for k, v := range s.Registry {
		out.Registry[k] = v
	}
	copy(out.Mask, s.Mask)
	return out
}

func assertBitPreserved(t *testing.T, label string, before, after []byte, idx int, bit byte) {
	t.Helper()
	if idx >= len(before) || idx >= len(after) {
		return
	}
	was := before[idx]&bit != 0
	now := after[idx]&bit != 0
	if was != now {
		t.Errorf("%s changed from %v to %v; unexplained bits must be preserved", label, was, now)
	}
}

func init() {
	// Keep the settle delay tolerable when the whole suite runs.
	if os.Getenv(systemTestEnv) == "1" {
		settleDelay = 1500 * time.Millisecond
	}
}

// TestMaskDerivationMatchesClearBits cross-checks the two independent descriptions of
// what the performance preset does to UserPreferencesMask: the recorded clear-mask, and
// the per-effect attribution table. They must agree, and neither may touch the two bits
// nothing was found to own.
func TestMaskDerivationMatchesClearBits(t *testing.T) {
	// Start from a mask with every bit set so any disagreement shows up.
	base := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	spi := map[string]int32{}
	for name := range maskBitFor {
		spi[name] = 1
	}
	for _, name := range bestPerformanceOff {
		spi[name] = 0
	}

	derived := maskFromSPI(spi, base)

	viaClearBits := make([]byte, len(base))
	copy(viaClearBits, base)
	for i := range viaClearBits {
		if i < len(maskClearBits) {
			viaClearBits[i] &^= maskClearBits[i]
		}
	}

	for i := range derived {
		if derived[i] != viaClearBits[i] {
			t.Errorf("byte[%d]: attribution table gives 0x%02X, recorded clear-mask gives 0x%02X",
				i, derived[i], viaClearBits[i])
		}
	}

	// The unexplained bits must survive both paths.
	if derived[2]&0x01 == 0 {
		t.Error("byte[2]:0x01 is unexplained and must be preserved by the derivation")
	}
	if derived[4]&0x10 == 0 {
		t.Error("byte[4]:0x10 is unexplained and must be preserved by the derivation")
	}
}

// TestMaskAttributionIsComplete pins the attribution established empirically, and pins
// that the two registry-only effects own no mask bit.
func TestMaskAttributionIsComplete(t *testing.T) {
	if len(maskBitFor) != 15 {
		t.Fatalf("attribution table has %d entries, want 15", len(maskBitFor))
	}
	for _, name := range []string{"DragFullWindows", "FontSmoothing"} {
		if _, ok := maskBitFor[name]; ok {
			t.Errorf("%s owns no mask bit; it lives only in the registry", name)
		}
	}
	// Every effect in the table must be a real effect.
	for name := range maskBitFor {
		if _, ok := findEffect(name); !ok {
			t.Errorf("attribution table references unknown effect %q", name)
		}
	}
}
