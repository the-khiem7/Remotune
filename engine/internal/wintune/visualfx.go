//go:build windows

package wintune

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/sys/windows/registry"
)

// effect is one Visual Effects value reachable through SystemParametersInfo.
type effect struct {
	name  string
	get   uint32
	set   uint32
	style spiStyle
}

// spiEffects are every per-effect value Remotune captures.
//
// Capturing more than Remotune changes is deliberate: the snapshot must describe the
// user's whole state, not only the part the performance preset touches.
var spiEffects = []effect{
	{"DragFullWindows", spiGetDragFullWindows, spiSetDragFullWindows, styleUIParam},
	{"FontSmoothing", spiGetFontSmoothing, spiSetFontSmoothing, styleUIParam},
	{"MenuAnimation", spiGetMenuAnimation, spiSetMenuAnimation, stylePvParam},
	{"ComboBoxAnimation", spiGetComboBoxAnimation, spiSetComboBoxAnimation, stylePvParam},
	{"ListBoxSmoothScrolling", spiGetListBoxSmoothScrolling, spiSetListBoxSmoothScrolling, stylePvParam},
	{"GradientCaptions", spiGetGradientCaptions, spiSetGradientCaptions, stylePvParam},
	{"KeyboardCues", spiGetKeyboardCues, spiSetKeyboardCues, stylePvParam},
	{"HotTracking", spiGetHotTracking, spiSetHotTracking, stylePvParam},
	{"MenuFade", spiGetMenuFade, spiSetMenuFade, stylePvParam},
	{"SelectionFade", spiGetSelectionFade, spiSetSelectionFade, stylePvParam},
	{"TooltipAnimation", spiGetTooltipAnimation, spiSetTooltipAnimation, stylePvParam},
	{"TooltipFade", spiGetTooltipFade, spiSetTooltipFade, stylePvParam},
	{"CursorShadow", spiGetCursorShadow, spiSetCursorShadow, stylePvParam},
	{"FlatMenu", spiGetFlatMenu, spiSetFlatMenu, stylePvParam},
	{"DropShadow", spiGetDropShadow, spiSetDropShadow, stylePvParam},
	{"UIEffects", spiGetUIEffects, spiSetUIEffects, stylePvParam},
	{"ClientAreaAnimation", spiGetClientAreaAnimation, spiSetClientAreaAnimation, stylePvParam},
}

// minAnimateKey is the SPI map entry for the ANIMATIONINFO-based value.
const minAnimateKey = "MinAnimate"

// bestPerformanceOff lists the effects the Windows performance preset turns off.
//
// The seven effects absent from this list are left alone on purpose: the real Windows
// preset does not touch them, so forcing them off would diverge from the behaviour
// Remotune claims to automate.
var bestPerformanceOff = []string{
	"DragFullWindows",
	"FontSmoothing",
	"MenuAnimation",
	"ComboBoxAnimation",
	"ListBoxSmoothScrolling",
	"SelectionFade",
	"TooltipAnimation",
	"CursorShadow",
	"DropShadow",
	"ClientAreaAnimation",
}

const (
	keyVisualEffects = `Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`
	keyDesktop       = `Control Panel\Desktop`
	keyWindowMetrics = `Control Panel\Desktop\WindowMetrics`
	keyAdvanced      = `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
	keyDWM           = `Software\Microsoft\Windows\DWM`

	maskValueName = "UserPreferencesMask"

	// labelKey is the preset label. Windows rewrites it asynchronously when individual
	// effects change, so it is written last and separately.
	labelKey = "VisualFXSetting"
)

// labelSettle is how long to let Windows finish re-labelling the configuration before
// writing the intended label over it.
var labelSettle = 400 * time.Millisecond

type regTarget struct {
	key  string
	name string
}

// veRegistry are the discrete values Remotune captures. Several have no SPI accessor
// at all, which is why SystemParametersInfo alone cannot describe the state.
var veRegistry = map[string]regTarget{
	"VisualFXSetting":               {keyVisualEffects, "VisualFXSetting"},
	"Desktop.DragFullWindows":       {keyDesktop, "DragFullWindows"},
	"Desktop.FontSmoothing":         {keyDesktop, "FontSmoothing"},
	"Desktop.FontSmoothingType":     {keyDesktop, "FontSmoothingType"},
	"Desktop.MenuShowDelay":         {keyDesktop, "MenuShowDelay"},
	"WindowMetrics.MinAnimate":      {keyWindowMetrics, "MinAnimate"},
	"Advanced.ListviewAlphaSelect":  {keyAdvanced, "ListviewAlphaSelect"},
	"Advanced.ListviewShadow":       {keyAdvanced, "ListviewShadow"},
	"Advanced.TaskbarAnimations":    {keyAdvanced, "TaskbarAnimations"},
	"Advanced.IconsOnly":            {keyAdvanced, "IconsOnly"},
	"DWM.EnableAeroPeek":            {keyDWM, "EnableAeroPeek"},
	"DWM.AlwaysHibernateThumbnails": {keyDWM, "AlwaysHibernateThumbnails"},
	"DWM.Composition":               {keyDWM, "Composition"},
}

// bestPerformanceRegistry are the registry values the preset writes, with the numeric
// target. Note IconsOnly rises to 1 while everything else falls to 0, so a blanket
// "zero everything" apply would be wrong.
//
// FontSmoothingType, MenuShowDelay and DWM.Composition are intentionally absent: the
// preset leaves them untouched.
var bestPerformanceRegistry = map[string]uint32{
	"VisualFXSetting":               2,
	"Desktop.DragFullWindows":       0,
	"Desktop.FontSmoothing":         0,
	"WindowMetrics.MinAnimate":      0,
	"Advanced.ListviewAlphaSelect":  0,
	"Advanced.ListviewShadow":       0,
	"Advanced.TaskbarAnimations":    0,
	"Advanced.IconsOnly":            1,
	"DWM.EnableAeroPeek":            0,
	"DWM.AlwaysHibernateThumbnails": 0,
}

// maskClearBits are the UserPreferencesMask bits the performance preset clears, per byte.
//
// Apply clears exactly these and preserves every other bit, which keeps the two bits
// that could not be attributed to any documented effect intact. Retained as an
// independent cross-check against the attribution table below.
var maskClearBits = [8]byte{0x0E, 0x2C, 0x04, 0x00, 0x02, 0x00, 0x00, 0x00}

// maskBit locates an effect inside UserPreferencesMask.
type maskBit struct {
	index int
	bit   byte
}

// maskBitFor maps each effect to the mask bit it owns, established by toggling every
// effect and observing which bit moved.
//
// Two set bits are deliberately absent because nothing was found to own them:
// byte[2]:0x01 and byte[4]:0x10. They are carried through untouched.
// DragFullWindows and FontSmoothing are absent because they own no mask bit at all and
// live only in the registry.
var maskBitFor = map[string]maskBit{
	"MenuAnimation":          {0, 0x02},
	"ComboBoxAnimation":      {0, 0x04},
	"ListBoxSmoothScrolling": {0, 0x08},
	"GradientCaptions":       {0, 0x10},
	"KeyboardCues":           {0, 0x20},
	"HotTracking":            {0, 0x80},
	"MenuFade":               {1, 0x02},
	"SelectionFade":          {1, 0x04},
	"TooltipAnimation":       {1, 0x08},
	"TooltipFade":            {1, 0x10},
	"CursorShadow":           {1, 0x20},
	"FlatMenu":               {2, 0x02},
	"DropShadow":             {2, 0x04},
	"UIEffects":              {3, 0x80},
	"ClientAreaAnimation":    {4, 0x02},
}

// maskFromSPI derives the mask implied by a set of per-effect values, starting from base
// so that unattributed bits survive.
//
// This keeps the two layers consistent by construction. Writing per-effect values while
// leaving the mask describing something else is the same defect class as the taskbar
// divergence: the live session and the persisted value disagree, and a reload can then
// undo the change.
func maskFromSPI(spi map[string]int32, base []byte) []byte {
	out := make([]byte, len(base))
	copy(out, base)
	for name, mb := range maskBitFor {
		v, ok := spi[name]
		if !ok || mb.index >= len(out) {
			continue
		}
		if v != 0 {
			out[mb.index] |= mb.bit
		} else {
			out[mb.index] &^= mb.bit
		}
	}
	return out
}

// VisualEffectsManager captures, applies and restores Windows Visual Effects state.
type VisualEffectsManager struct{}

// Snapshot captures all three layers of the current state.
func (m *VisualEffectsManager) Snapshot() (*VisualEffectsSnapshot, error) {
	spi := make(map[string]int32, len(spiEffects)+1)
	for _, e := range spiEffects {
		v, err := spiGetBool(e.get)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.name, err)
		}
		spi[e.name] = v
	}
	ma, err := spiGetMinAnimate()
	if err != nil {
		return nil, err
	}
	spi[minAnimateKey] = ma

	reg := make(map[string]RegValue, len(veRegistry))
	for logical, t := range veRegistry {
		v, err := readRegValue(t)
		if err != nil {
			// A value that is genuinely absent is skipped rather than invented.
			continue
		}
		reg[logical] = v
	}

	mask, err := readMask()
	if err != nil {
		return nil, err
	}

	return &VisualEffectsSnapshot{SPI: spi, Registry: reg, Mask: mask}, nil
}

// GetCurrentState is an alias for Snapshot used where no mutation is intended.
func (m *VisualEffectsManager) GetCurrentState() (*VisualEffectsSnapshot, error) {
	return m.Snapshot()
}

// convergeAttempts bounds how many times a write sequence is re-asserted before the
// transition is reported as failed.
//
// One pass is not sufficient. The shell reloads user settings asynchronously, so a write
// that succeeded can still be observed as not-yet-applied, or be transiently overwritten
// by a reload that was already in flight. Rather than trusting the write or sleeping
// longer and hoping, each pass re-asserts only the values that do not yet match and the
// loop exits as soon as the observed state equals the target.
const convergeAttempts = 4

// BestPerformanceTarget computes the state Windows' "Adjust for best performance" would
// produce from the given starting state.
//
// It is expressed as a transformation of the current state rather than as a stored
// target, so no machine-specific value from another system is ever propagated.
func BestPerformanceTarget(before *VisualEffectsSnapshot) *VisualEffectsSnapshot {
	target := &VisualEffectsSnapshot{
		SPI:      make(map[string]int32, len(before.SPI)),
		Registry: make(map[string]RegValue, len(before.Registry)),
		Mask:     make([]byte, len(before.Mask)),
	}
	// start from the current state so untouched values are carried through unchanged
	for k, v := range before.SPI {
		target.SPI[k] = v
	}
	for k, v := range before.Registry {
		target.Registry[k] = v
	}
	copy(target.Mask, before.Mask)

	for _, name := range bestPerformanceOff {
		target.SPI[name] = 0
	}
	target.SPI[minAnimateKey] = 0

	for logical, n := range bestPerformanceRegistry {
		kind := RegKindDWord
		if cur, ok := before.Registry[logical]; ok {
			kind = cur.Kind
		}
		if kind == RegKindString {
			target.Registry[logical] = RegValue{Kind: RegKindString, Str: strconv.FormatUint(uint64(n), 10)}
		} else {
			target.Registry[logical] = RegValue{Kind: RegKindDWord, DWord: n}
		}
	}

	// Derive the mask from the per-effect targets so the two layers cannot disagree.
	// For this transformation the result is identical to clearing maskClearBits, which
	// TestMaskDerivationMatchesClearBits asserts.
	target.Mask = maskFromSPI(target.SPI, before.Mask)
	return target
}

// writeState writes every value in target, in the order established empirically.
//
// SPI first: a SystemParametersInfo write has side effects on values Remotune also
// records, so the explicit registry and mask writes must follow it to win.
// Mask last: it is the only step that can carry the bits SPI cannot express.
func writeState(target *VisualEffectsSnapshot, only map[string]bool) error {
	want := func(key string) bool { return only == nil || only[key] }

	for _, e := range spiEffects {
		v, ok := target.SPI[e.name]
		if !ok || !want("spi."+e.name) {
			continue
		}
		if err := spiSetBool(e.set, e.style, v); err != nil {
			return err
		}
	}
	if v, ok := target.SPI[minAnimateKey]; ok && want("spi."+minAnimateKey) {
		if err := spiSetMinAnimate(v); err != nil {
			return err
		}
	}

	// Everything except the preset label.
	for logical, v := range target.Registry {
		t, ok := veRegistry[logical]
		if !ok || logical == labelKey || !want("registry."+logical) {
			continue
		}
		if err := writeRegValue(t, v); err != nil {
			return fmt.Errorf("write %s: %w", logical, err)
		}
	}

	if len(target.Mask) > 0 && want(maskValueName) {
		if err := writeMask(target.Mask); err != nil {
			return err
		}
	}

	// The preset label goes LAST and on its own.
	//
	// Changing individual effects makes Windows re-label the configuration as Custom, and
	// it does so asynchronously. Writing the label before or alongside the effect writes
	// meant Windows overwrote it immediately afterwards, so the value never converged no
	// matter how many times it was re-asserted.
	if v, ok := target.Registry[labelKey]; ok && want("registry."+labelKey) {
		time.Sleep(labelSettle)
		if err := writeRegValue(veRegistry[labelKey], v); err != nil {
			return fmt.Errorf("write %s: %w", labelKey, err)
		}
	}
	return nil
}

// convergeTo drives the system to target, re-asserting only what has not landed yet.
func (m *VisualEffectsManager) convergeTo(target *VisualEffectsSnapshot) (*VisualEffectsSnapshot, []string, error) {
	var observed *VisualEffectsSnapshot
	var diff []string
	only := map[string]bool(nil)

	for attempt := 0; attempt < convergeAttempts; attempt++ {
		if err := writeState(target, only); err != nil {
			return nil, nil, err
		}
		// No global WM_SETTINGCHANGE broadcast inside the loop.
		//
		// Each per-effect write already notifies via SPIF_SENDCHANGE. An extra global
		// broadcast made the shell reload user settings from the registry part-way through
		// the sequence, pulling the live session back to whatever the mask said at that
		// instant and undoing writes that had already landed.
		time.Sleep(settleVisualEffects)

		var err error
		observed, err = m.Snapshot()
		if err != nil {
			return nil, nil, err
		}
		diff = DiffVisualEffects(target, observed)
		if len(diff) == 0 {
			// One broadcast at the end, once the persisted state already matches the
			// target, so a reload can only confirm it.
			broadcastSettingChange("WindowMetrics")
			return observed, nil, nil
		}
		// Next pass re-asserts only the values still diverging.
		only = make(map[string]bool, len(diff))
		for _, k := range diff {
			only[k] = true
		}
	}
	return observed, diff, nil
}

// ApplyBestPerformance applies the Windows performance preset as a transformation.
//
// It changes only what the real preset changes and never replays a captured target
// state, because a captured Best Performance snapshot carries machine-specific values
// that must not be pushed onto other machines.
func (m *VisualEffectsManager) ApplyBestPerformance() (CategoryResult, error) {
	res := CategoryResult{Category: CategoryVisualEffects}

	before, err := m.Snapshot()
	if err != nil {
		res.Err = err
		return res, err
	}

	target := BestPerformanceTarget(before)
	observed, diff, err := m.convergeTo(target)
	if err != nil {
		res.Err = err
		return res, err
	}
	res.Changed = !visualEffectsEqual(before, observed)
	if len(diff) > 0 {
		res.Err = fmt.Errorf("apply did not converge after %d attempts, %d value(s) differ: %v",
			convergeAttempts, len(diff), diff)
		return res, res.Err
	}
	if err := verifyBestPerformance(observed); err != nil {
		res.Err = err
		return res, err
	}
	res.Verified = true
	return res, nil
}

// Restore returns the system to a captured state, converging until the observed state
// matches the snapshot exactly.
//
// A nil or incomplete snapshot is never guessed around: the caller is told no restorable
// state exists instead of being handed an invented baseline.
func (m *VisualEffectsManager) Restore(s *VisualEffectsSnapshot) (CategoryResult, error) {
	res := CategoryResult{Category: CategoryVisualEffects}
	if s == nil {
		res.Err = errors.New("no visual effects snapshot to restore")
		return res, res.Err
	}
	if len(s.Mask) == 0 || len(s.SPI) == 0 || len(s.Registry) == 0 {
		res.Err = errors.New("visual effects snapshot is incomplete; refusing to guess")
		return res, res.Err
	}

	before, err := m.Snapshot()
	if err != nil {
		res.Err = err
		return res, err
	}

	// Repair any disagreement between the snapshot's per-effect values and its mask
	// before writing. For a snapshot captured from a real system this is a no-op because
	// the layers already agree; it only matters for a hand-built target.
	target := &VisualEffectsSnapshot{
		SPI:      s.SPI,
		Registry: s.Registry,
		Mask:     maskFromSPI(s.SPI, s.Mask),
	}

	observed, diff, err := m.convergeTo(target)
	if err != nil {
		res.Err = err
		return res, err
	}
	res.Changed = !visualEffectsEqual(before, observed)
	if len(diff) > 0 {
		res.Err = fmt.Errorf("restore did not converge after %d attempts, %d value(s) differ: %v",
			convergeAttempts, len(diff), diff)
		return res, res.Err
	}
	res.Verified = true
	return res, nil
}

func findEffect(name string) (effect, bool) {
	for _, e := range spiEffects {
		if e.name == name {
			return e, true
		}
	}
	return effect{}, false
}

// verifyBestPerformance confirms the observable outcome of an apply: every effect the
// preset turns off reads as off, and no untouched effect was disturbed.
func verifyBestPerformance(s *VisualEffectsSnapshot) error {
	for _, name := range bestPerformanceOff {
		if v, ok := s.SPI[name]; ok && v != 0 {
			return fmt.Errorf("%s is %d after apply, want 0", name, v)
		}
	}
	if v, ok := s.SPI[minAnimateKey]; ok && v != 0 {
		return fmt.Errorf("MinAnimate is %d after apply, want 0", v)
	}
	if v, ok := s.Registry["VisualFXSetting"]; ok && v.DWord != 2 {
		return fmt.Errorf("VisualFXSetting is %s after apply, want 2", v)
	}
	if v, ok := s.Registry["Advanced.IconsOnly"]; ok && v.DWord != 1 {
		return fmt.Errorf("Advanced.IconsOnly is %s after apply, want 1", v)
	}
	return nil
}

// DiffVisualEffects lists the logical keys whose values differ between two states.
func DiffVisualEffects(a, b *VisualEffectsSnapshot) []string {
	var diff []string
	if a == nil || b == nil {
		return []string{"<nil snapshot>"}
	}
	for k, av := range a.SPI {
		if bv, ok := b.SPI[k]; !ok || av != bv {
			diff = append(diff, "spi."+k)
		}
	}
	for k, av := range a.Registry {
		bv, ok := b.Registry[k]
		if !ok || !av.Equal(bv) {
			diff = append(diff, "registry."+k)
		}
	}
	if len(a.Mask) != len(b.Mask) {
		diff = append(diff, maskValueName)
	} else {
		for i := range a.Mask {
			if a.Mask[i] != b.Mask[i] {
				diff = append(diff, maskValueName)
				break
			}
		}
	}
	return diff
}

func visualEffectsEqual(a, b *VisualEffectsSnapshot) bool {
	return len(DiffVisualEffects(a, b)) == 0
}

func readMask() ([]byte, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyDesktop, registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", keyDesktop, err)
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue(maskValueName)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", maskValueName, err)
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

func writeMask(b []byte) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyDesktop, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open %s for write: %w", keyDesktop, err)
	}
	defer k.Close()
	return k.SetBinaryValue(maskValueName, b)
}

// readRegValue reads a value preserving whether it is stored as a string or a dword.
func readRegValue(t regTarget) (RegValue, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, t.key, registry.QUERY_VALUE)
	if err != nil {
		return RegValue{}, err
	}
	defer k.Close()

	if n, _, err := k.GetIntegerValue(t.name); err == nil {
		return RegValue{Kind: RegKindDWord, DWord: uint32(n)}, nil
	}
	s, _, err := k.GetStringValue(t.name)
	if err != nil {
		return RegValue{}, err
	}
	return RegValue{Kind: RegKindString, Str: s}, nil
}

func writeRegValue(t regTarget, v RegValue) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, t.key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if v.Kind == RegKindString {
		return k.SetStringValue(t.name, v.Str)
	}
	return k.SetDWordValue(t.name, v.DWord)
}

// writeRegNumeric writes a numeric target while preserving the value's existing type,
// so a REG_SZ holding digits is not silently converted to REG_DWORD.
func writeRegNumeric(t regTarget, n uint32) error {
	existing, err := readRegValue(t)
	kind := RegKindDWord
	if err == nil {
		kind = existing.Kind
	}
	if kind == RegKindString {
		return writeRegValue(t, RegValue{Kind: RegKindString, Str: strconv.FormatUint(uint64(n), 10)})
	}
	return writeRegValue(t, RegValue{Kind: RegKindDWord, DWord: n})
}

// settleVisualEffects is how long to wait after the broadcast before re-reading.
// Measured stable at 800 ms; a shorter wait produced intermittent reads that caught the
// shell mid-reload.
var settleVisualEffects = 800 * time.Millisecond
