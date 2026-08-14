//go:build windows

// Package wintune contains the Windows adapters for Remotune's tuning engine.
//
// It deliberately has no dependency on Wails or on any UI framework, so it can be
// built and tested headlessly and later migrated into the CLI-generated application
// project without redesign.
package wintune

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	shell32 = windows.NewLazySystemDLL("shell32.dll")
	procSPI = user32.NewProc("SystemParametersInfoW")
	procSMT = user32.NewProc("SendMessageTimeoutW")
	procABM = shell32.NewProc("SHAppBarMessage")
	procGSM = user32.NewProc("GetSystemMetrics")
)

// SystemParametersInfo flags.
const (
	spifUpdateIniFile = 0x01
	spifSendChange    = 0x02

	// spifWrite is the flag combination used for per-effect writes.
	//
	// SPIF_UPDATEINIFILE is deliberately NOT included. That flag makes each write persist
	// itself by read-modify-writing the shared UserPreferencesMask byte, and consecutive
	// writes to effects that share a byte were measured racing and losing each other's
	// bits. Worse, a later write could resurrect a byte from a stale copy and undo an
	// explicit mask write, so the two layers never converged.
	//
	// The adapter therefore splits the responsibilities: SystemParametersInfo changes only
	// the live session, and persistence is done by writing the complete mask and the
	// discrete registry values explicitly. This is the same rule the taskbar adapter
	// follows after its divergence defect: write each layer deliberately instead of hoping
	// one propagates to the other.
	spifWrite = spifSendChange
)

// SystemParametersInfo actions used by the Visual Effects adapter.
const (
	spiGetDragFullWindows        = 0x0026
	spiSetDragFullWindows        = 0x0025
	spiGetFontSmoothing          = 0x004A
	spiSetFontSmoothing          = 0x004B
	spiGetAnimation              = 0x0048
	spiSetAnimation              = 0x0049
	spiGetWorkArea               = 0x0030
	spiGetMenuAnimation          = 0x1002
	spiSetMenuAnimation          = 0x1003
	spiGetComboBoxAnimation      = 0x1004
	spiSetComboBoxAnimation      = 0x1005
	spiGetListBoxSmoothScrolling = 0x1006
	spiSetListBoxSmoothScrolling = 0x1007
	spiGetGradientCaptions       = 0x1008
	spiSetGradientCaptions       = 0x1009
	spiGetKeyboardCues           = 0x100A
	spiSetKeyboardCues           = 0x100B
	spiGetHotTracking            = 0x100E
	spiSetHotTracking            = 0x100F
	spiGetMenuFade               = 0x1012
	spiSetMenuFade               = 0x1013
	spiGetSelectionFade          = 0x1014
	spiSetSelectionFade          = 0x1015
	spiGetTooltipAnimation       = 0x1016
	spiSetTooltipAnimation       = 0x1017
	spiGetTooltipFade            = 0x1018
	spiSetTooltipFade            = 0x1019
	spiGetCursorShadow           = 0x101A
	spiSetCursorShadow           = 0x101B
	spiGetFlatMenu               = 0x1022
	spiSetFlatMenu               = 0x1023
	spiGetDropShadow             = 0x1024
	spiSetDropShadow             = 0x1025
	spiGetUIEffects              = 0x103E
	spiSetUIEffects              = 0x103F
	spiGetClientAreaAnimation    = 0x1042
	spiSetClientAreaAnimation    = 0x1043
)

// Appbar messages and state bits.
const (
	abmGetState      = 0x00000004
	abmGetTaskbarPos = 0x00000005
	abmSetState      = 0x0000000A

	// ABSAutoHide is the ABS_AUTOHIDE bit. It is the only bit Remotune ever changes.
	ABSAutoHide = 0x00000001
	// ABSAlwaysOnTop is preserved, never written by Remotune.
	ABSAlwaysOnTop = 0x00000002
)

const (
	smCXScreen = 0
	smCYScreen = 1

	wmSettingChange = 0x001A
	hwndBroadcast   = 0xFFFF
	smtoAbortIfHung = 0x0002
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type animationInfo struct {
	CbSize      uint32
	IMinAnimate int32
}

type appBarData struct {
	CbSize           uint32
	HWnd             windows.Handle
	UCallbackMessage uint32
	UEdge            uint32
	Rc               rect
	LParam           int32
	_                int32 // pad so the struct matches the 64-bit layout
}

// spiStyle records how a given SystemParametersInfo action carries its value.
//
// This distinction is load-bearing: passing the value in the wrong slot silently
// fails or writes the wrong setting.
type spiStyle int

const (
	// styleUIParam carries the value in uiParam with pvParam nil.
	styleUIParam spiStyle = iota
	// stylePvParam carries the value in pvParam as a BOOL.
	stylePvParam
)

// spiGetBool reads a boolean-style setting. Every documented get action writes into
// pvParam, regardless of how its matching set action carries the value.
func spiGetBool(action uint32) (int32, error) {
	var v int32
	r, _, err := procSPI.Call(
		uintptr(action),
		0,
		uintptr(unsafe.Pointer(&v)),
		0,
	)
	if r == 0 {
		return 0, fmt.Errorf("SystemParametersInfo get 0x%04X: %w", action, err)
	}
	return v, nil
}

func spiSetBool(action uint32, style spiStyle, value int32) error {
	var r uintptr
	var err error
	switch style {
	case styleUIParam:
		r, _, err = procSPI.Call(uintptr(action), uintptr(value), 0, spifWrite)
	default:
		r, _, err = procSPI.Call(uintptr(action), 0, uintptr(value), spifWrite)
	}
	if r == 0 {
		return fmt.Errorf("SystemParametersInfo set 0x%04X: %w", action, err)
	}
	return nil
}

func spiGetMinAnimate() (int32, error) {
	ai := animationInfo{CbSize: uint32(unsafe.Sizeof(animationInfo{}))}
	r, _, err := procSPI.Call(
		uintptr(spiGetAnimation),
		uintptr(ai.CbSize),
		uintptr(unsafe.Pointer(&ai)),
		0,
	)
	if r == 0 {
		return 0, fmt.Errorf("SystemParametersInfo SPI_GETANIMATION: %w", err)
	}
	return ai.IMinAnimate, nil
}

func spiSetMinAnimate(value int32) error {
	ai := animationInfo{
		CbSize:      uint32(unsafe.Sizeof(animationInfo{})),
		IMinAnimate: value,
	}
	r, _, err := procSPI.Call(
		uintptr(spiSetAnimation),
		uintptr(ai.CbSize),
		uintptr(unsafe.Pointer(&ai)),
		spifWrite,
	)
	if r == 0 {
		return fmt.Errorf("SystemParametersInfo SPI_SETANIMATION: %w", err)
	}
	return nil
}

// appBarGetState returns the raw ABM_GETSTATE bits.
func appBarGetState() uint32 {
	d := appBarData{CbSize: uint32(unsafe.Sizeof(appBarData{}))}
	r, _, _ := procABM.Call(uintptr(abmGetState), uintptr(unsafe.Pointer(&d)))
	return uint32(r)
}

// appBarSetState writes the full state word. Callers must preserve unrelated bits.
func appBarSetState(state uint32) error {
	d := appBarData{
		CbSize: uint32(unsafe.Sizeof(appBarData{})),
		LParam: int32(state),
	}
	r, _, err := procABM.Call(uintptr(abmSetState), uintptr(unsafe.Pointer(&d)))
	if r == 0 {
		// SHAppBarMessage returns FALSE for ABM_SETSTATE on some builds even on
		// success, so this is reported but not treated as fatal by callers that
		// verify the observable outcome instead.
		return fmt.Errorf("SHAppBarMessage ABM_SETSTATE: %w", err)
	}
	return nil
}

// broadcastSettingChange notifies the shell that user settings changed.
func broadcastSettingChange(area string) {
	p, err := windows.UTF16PtrFromString(area)
	if err != nil {
		return
	}
	var result uintptr
	procSMT.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(p)),
		smtoAbortIfHung,
		1000,
		uintptr(unsafe.Pointer(&result)),
	)
}

func getSystemMetrics(index int32) int32 {
	r, _, _ := procGSM.Call(uintptr(index))
	return int32(r)
}

// primaryWorkArea returns the primary monitor work area.
func primaryWorkArea() (rect, error) {
	var rc rect
	r, _, err := procSPI.Call(
		uintptr(spiGetWorkArea),
		0,
		uintptr(unsafe.Pointer(&rc)),
		0,
	)
	if r == 0 {
		return rc, fmt.Errorf("SystemParametersInfo SPI_GETWORKAREA: %w", err)
	}
	return rc, nil
}
