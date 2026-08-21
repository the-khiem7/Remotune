//go:build windows

package wintune

import (
	"fmt"
	"strconv"
)

type VisualEffectsProfile string

const (
	ProfileWindowsChoose   VisualEffectsProfile = "windowsChoose"
	ProfileBestAppearance  VisualEffectsProfile = "bestAppearance"
	ProfileBestPerformance VisualEffectsProfile = "bestPerformance"
	ProfileCustom          VisualEffectsProfile = "custom"
)

func (p VisualEffectsProfile) Valid() bool {
	switch p {
	case ProfileWindowsChoose, ProfileBestAppearance, ProfileBestPerformance, ProfileCustom:
		return true
	default:
		return false
	}
}

func EffectNames() []string {
	return []string{"AnimateControls", "AnimateWindows", "TaskbarAnimations", "EnablePeek", "MenuAnimation", "TooltipAnimation", "SelectionFade", "SaveTaskbarThumbnails", "CursorShadow", "DropShadow", "ShowThumbnails", "TranslucentSelection", "DragFullWindows", "ComboBoxAnimation", "FontSmoothing", "ListBoxSmoothScrolling", "IconLabelShadow"}
}

func ProfileTarget(before *VisualEffectsSnapshot, profile VisualEffectsProfile, custom map[string]bool) (*VisualEffectsSnapshot, error) {
	if before == nil || len(before.SPI) == 0 || len(before.Registry) == 0 || len(before.Mask) == 0 {
		return nil, fmt.Errorf("cannot compile %s without a complete current visual effects state", profile)
	}
	if !profile.Valid() {
		return nil, fmt.Errorf("unsupported visual effects profile %q", profile)
	}
	if profile == ProfileBestPerformance {
		return BestPerformanceTarget(before), nil
	}
	target := cloneVisualEffects(before)
	label := uint32(0)
	switch profile {
	case ProfileWindowsChoose:
		label = 0
	case ProfileBestAppearance:
		label = 1
		for _, name := range EffectNames() {
			applyCustomEffect(target, name, true)
		}
	case ProfileCustom:
		label = 3
		for _, name := range EffectNames() {
			applyCustomEffect(target, name, custom[name])
		}
	}
	setNumeric(target, labelKey, int32(label))
	target.Mask = maskFromSPI(target.SPI, before.Mask)
	return target, nil
}

func applyCustomEffect(target *VisualEffectsSnapshot, name string, enabled bool) {
	value := int32(0)
	if enabled {
		value = 1
	}
	switch name {
	case "AnimateControls":
		target.SPI["UIEffects"] = value
	case "AnimateWindows":
		target.SPI[minAnimateKey] = value
	case "TaskbarAnimations":
		setNumeric(target, "Advanced.TaskbarAnimations", value)
	case "EnablePeek":
		setNumeric(target, "DWM.EnableAeroPeek", value)
	case "MenuAnimation":
		target.SPI["MenuAnimation"] = value
	case "TooltipAnimation":
		target.SPI["TooltipAnimation"] = value
	case "SelectionFade":
		target.SPI["SelectionFade"] = value
	case "SaveTaskbarThumbnails":
		setNumeric(target, "DWM.AlwaysHibernateThumbnails", value)
	case "CursorShadow":
		target.SPI["CursorShadow"] = value
	case "DropShadow":
		target.SPI["DropShadow"] = value
	case "ShowThumbnails":
		setNumeric(target, "Advanced.IconsOnly", 1-value)
	case "TranslucentSelection":
		setNumeric(target, "Advanced.ListviewAlphaSelect", value)
	case "DragFullWindows":
		target.SPI["DragFullWindows"] = value
		setNumeric(target, "Desktop.DragFullWindows", value)
	case "ComboBoxAnimation":
		target.SPI["ComboBoxAnimation"] = value
	case "FontSmoothing":
		target.SPI["FontSmoothing"] = value
		setNumeric(target, "Desktop.FontSmoothing", value*2)
	case "ListBoxSmoothScrolling":
		target.SPI["ListBoxSmoothScrolling"] = value
	case "IconLabelShadow":
		setNumeric(target, "Advanced.ListviewShadow", value)
	}
}

func cloneVisualEffects(source *VisualEffectsSnapshot) *VisualEffectsSnapshot {
	target := &VisualEffectsSnapshot{SPI: map[string]int32{}, Registry: map[string]RegValue{}, Mask: append([]byte(nil), source.Mask...)}
	for name, value := range source.SPI {
		target.SPI[name] = value
	}
	for name, value := range source.Registry {
		target.Registry[name] = value
	}
	return target
}

func setNumeric(target *VisualEffectsSnapshot, logical string, value int32) {
	kind := RegKindDWord
	if existing, ok := target.Registry[logical]; ok {
		kind = existing.Kind
	}
	if kind == RegKindString {
		target.Registry[logical] = RegValue{Kind: kind, Str: strconv.FormatInt(int64(value), 10)}
		return
	}
	target.Registry[logical] = RegValue{Kind: kind, DWord: uint32(value)}
}
