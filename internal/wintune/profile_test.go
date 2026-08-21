//go:build windows

package wintune

import "testing"

func TestProfileTargetPreservesWindowsChooseValuesAndChangesOnlyLabel(t *testing.T) {
	before := &VisualEffectsSnapshot{
		SPI:      map[string]int32{"MenuAnimation": 1, minAnimateKey: 1},
		Registry: map[string]RegValue{"VisualFXSetting": {Kind: RegKindDWord, DWord: 1}},
		Mask:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	target, err := ProfileTarget(before, ProfileWindowsChoose, nil)
	if err != nil {
		t.Fatalf("ProfileTarget: %v", err)
	}
	if got := target.SPI["MenuAnimation"]; got != 1 {
		t.Fatalf("MenuAnimation = %d, want preserved value 1", got)
	}
	if got := target.Registry["VisualFXSetting"].DWord; got != 0 {
		t.Fatalf("VisualFXSetting = %d, want 0", got)
	}
}

func TestBestAppearanceTargetEnablesVisualEffectsAndPreservesMaskUnknownBits(t *testing.T) {
	before := &VisualEffectsSnapshot{
		SPI:      map[string]int32{},
		Registry: map[string]RegValue{"VisualFXSetting": {Kind: RegKindDWord, DWord: 2}},
		Mask:     []byte{0, 0, 1, 0, 0x10, 0, 0, 0},
	}
	before.SPI["UIEffects"] = 0
	before.SPI[minAnimateKey] = 0
	before.SPI["MenuAnimation"] = 0
	before.SPI["TooltipAnimation"] = 0
	before.SPI["SelectionFade"] = 0
	before.SPI["CursorShadow"] = 0
	before.SPI["DropShadow"] = 0
	before.SPI["DragFullWindows"] = 0
	before.SPI["ComboBoxAnimation"] = 0
	before.SPI["FontSmoothing"] = 0
	before.SPI["ListBoxSmoothScrolling"] = 0
	target, err := ProfileTarget(before, ProfileBestAppearance, nil)
	if err != nil {
		t.Fatalf("ProfileTarget: %v", err)
	}
	for _, name := range []string{"UIEffects", minAnimateKey, "MenuAnimation", "TooltipAnimation", "SelectionFade", "CursorShadow", "DropShadow", "DragFullWindows", "ComboBoxAnimation", "FontSmoothing", "ListBoxSmoothScrolling"} {
		if target.SPI[name] != 1 {
			t.Fatalf("%s = %d, want 1", name, target.SPI[name])
		}
	}
	if target.Mask[2]&0x01 == 0 || target.Mask[4]&0x10 == 0 {
		t.Fatal("unknown mask bits were not preserved")
	}
}
