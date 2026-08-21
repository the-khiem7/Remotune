//go:build windows

package wintune

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/sys/windows/registry"
)

type effect struct {
	name  string
	get   uint32
	set   uint32
	style spiStyle
}

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

const minAnimateKey = "MinAnimate"

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
	labelKey      = "VisualFXSetting"
)

var labelSettle = 400 * time.Millisecond

type regTarget struct {
	key  string
	name string
}

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
var maskClearBits = [8]byte{0x0E, 0x2C, 0x04, 0x00, 0x02, 0x00, 0x00, 0x00}

type maskBit struct {
	index int
	bit   byte
}

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

type VisualEffectsManager struct{}

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
func (m *VisualEffectsManager) GetCurrentState() (*VisualEffectsSnapshot, error) {
	return m.Snapshot()
}

const convergeAttempts = 4

func BestPerformanceTarget(before *VisualEffectsSnapshot) *VisualEffectsSnapshot {
	target := &VisualEffectsSnapshot{
		SPI:      make(map[string]int32, len(before.SPI)),
		Registry: make(map[string]RegValue, len(before.Registry)),
		Mask:     make([]byte, len(before.Mask)),
	}
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
	target.Mask = maskFromSPI(target.SPI, before.Mask)
	return target
}
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
	if v, ok := target.Registry[labelKey]; ok && want("registry."+labelKey) {
		time.Sleep(labelSettle)
		if err := writeRegValue(veRegistry[labelKey], v); err != nil {
			return fmt.Errorf("write %s: %w", labelKey, err)
		}
	}
	return nil
}
func (m *VisualEffectsManager) convergeTo(target *VisualEffectsSnapshot) (*VisualEffectsSnapshot, []string, error) {
	var observed *VisualEffectsSnapshot
	var diff []string
	only := map[string]bool(nil)

	for attempt := 0; attempt < convergeAttempts; attempt++ {
		if err := writeState(target, only); err != nil {
			return nil, nil, err
		}
		time.Sleep(settleVisualEffects)

		var err error
		observed, err = m.Snapshot()
		if err != nil {
			return nil, nil, err
		}
		diff = DiffVisualEffects(target, observed)
		if len(diff) == 0 {
			broadcastSettingChange("WindowMetrics")
			return observed, nil, nil
		}
		only = make(map[string]bool, len(diff))
		for _, k := range diff {
			only[k] = true
		}
	}
	return observed, diff, nil
}
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

var settleVisualEffects = 800 * time.Millisecond
