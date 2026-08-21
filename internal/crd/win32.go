//go:build windows

package crd

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	wevtapi  = windows.NewLazySystemDLL("wevtapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procEvtQuery          = wevtapi.NewProc("EvtQuery")
	procEvtNext           = wevtapi.NewProc("EvtNext")
	procEvtRender         = wevtapi.NewProc("EvtRender")
	procEvtClose          = wevtapi.NewProc("EvtClose")
	procEvtCreateBookmark = wevtapi.NewProc("EvtCreateBookmark")
	procEvtUpdateBookmark = wevtapi.NewProc("EvtUpdateBookmark")
	procEvtSubscribe      = wevtapi.NewProc("EvtSubscribe")

	procCreateEventW        = kernel32.NewProc("CreateEventW")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procResetEvent          = kernel32.NewProc("ResetEvent")
)

const (
	waitObject0 = 0x00000000
)

func waitForSignal(h windows.Handle, timeoutMs uint32) bool {
	r, _, _ := procWaitForSingleObject.Call(uintptr(h), uintptr(timeoutMs))
	return r == waitObject0
}
func resetSignal(h windows.Handle) {
	procResetEvent.Call(uintptr(h))
}

const (
	evtQueryChannelPath      = 0x1
	evtQueryReverseDirection = 0x200
)
const (
	evtRenderEventXml    = 1
	evtRenderBookmarkXml = 2
)
const (
	evtSubscribeToFutureEvents      = 1
	evtSubscribeStartAtOldestRecord = 2
	evtSubscribeStartAfterBookmark  = 3
)
const errorNoMoreItems = syscall.Errno(259)

type evtHandle uintptr

func (h evtHandle) valid() bool { return h != 0 }

func evtClose(h evtHandle) {
	if h == 0 {
		return
	}
	procEvtClose.Call(uintptr(h))
}
func evtQuery(channel, xpath string, reverse bool) (evtHandle, error) {
	cPath, err := windows.UTF16PtrFromString(channel)
	if err != nil {
		return 0, err
	}
	cQuery, err := windows.UTF16PtrFromString(xpath)
	if err != nil {
		return 0, err
	}
	flags := uintptr(evtQueryChannelPath)
	if reverse {
		flags |= evtQueryReverseDirection
	}
	r, _, err := procEvtQuery.Call(0, uintptr(unsafe.Pointer(cPath)), uintptr(unsafe.Pointer(cQuery)), flags)
	if r == 0 {
		return 0, fmt.Errorf("EvtQuery: %w", err)
	}
	return evtHandle(r), nil
}
func evtNext(resultSet evtHandle, out []evtHandle, timeoutMs uint32) ([]evtHandle, error) {
	if len(out) == 0 {
		return nil, nil
	}
	var returned uint32
	r, _, err := procEvtNext.Call(
		uintptr(resultSet),
		uintptr(len(out)),
		uintptr(unsafe.Pointer(&out[0])),
		uintptr(timeoutMs),
		0,
		uintptr(unsafe.Pointer(&returned)),
	)
	if r == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno == errorNoMoreItems {
			return nil, nil
		}
		return nil, fmt.Errorf("EvtNext: %w", err)
	}
	return out[:returned], nil
}
func evtRenderEventXML(event evtHandle) (string, error) {
	return evtRenderXML(event, evtRenderEventXml)
}

func evtRenderBookmarkXML(bookmark evtHandle) (string, error) {
	return evtRenderXML(bookmark, evtRenderBookmarkXml)
}

func evtRenderXML(handle evtHandle, flags uint32) (string, error) {
	var used, propCount uint32
	procEvtRender.Call(0, uintptr(handle), uintptr(flags), 0, 0, uintptr(unsafe.Pointer(&used)), uintptr(unsafe.Pointer(&propCount)))

	if used == 0 {
		return "", fmt.Errorf("EvtRender: could not determine buffer size")
	}
	buf := make([]uint16, used/2+1)
	r, _, err := procEvtRender.Call(
		0, uintptr(handle), uintptr(flags),
		uintptr(len(buf)*2), uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&used)), uintptr(unsafe.Pointer(&propCount)),
	)
	if r == 0 {
		return "", fmt.Errorf("EvtRender: %w", err)
	}
	return windows.UTF16ToString(buf), nil
}
func evtCreateBookmark(xml string) (evtHandle, error) {
	var ptr *uint16
	if xml != "" {
		p, err := windows.UTF16PtrFromString(xml)
		if err != nil {
			return 0, err
		}
		ptr = p
	}
	r, _, err := procEvtCreateBookmark.Call(uintptr(unsafe.Pointer(ptr)))
	if r == 0 {
		return 0, fmt.Errorf("EvtCreateBookmark: %w", err)
	}
	return evtHandle(r), nil
}
func evtUpdateBookmark(bookmark, event evtHandle) error {
	r, _, err := procEvtUpdateBookmark.Call(uintptr(bookmark), uintptr(event))
	if r == 0 {
		return fmt.Errorf("EvtUpdateBookmark: %w", err)
	}
	return nil
}
func evtSubscribe(channel, xpath string, bookmark evtHandle, signalEvent windows.Handle, flags uint32) (evtHandle, error) {
	cPath, err := windows.UTF16PtrFromString(channel)
	if err != nil {
		return 0, err
	}
	cQuery, err := windows.UTF16PtrFromString(xpath)
	if err != nil {
		return 0, err
	}
	r, _, err := procEvtSubscribe.Call(
		0, uintptr(signalEvent),
		uintptr(unsafe.Pointer(cPath)),
		uintptr(unsafe.Pointer(cQuery)),
		uintptr(bookmark),
		0, 0,
		uintptr(flags),
	)
	if r == 0 {
		return 0, fmt.Errorf("EvtSubscribe: %w", err)
	}
	return evtHandle(r), nil
}
func createManualResetEvent() (windows.Handle, error) {
	r, _, err := procCreateEventW.Call(0, 1, 1, 0)
	if r == 0 {
		return 0, fmt.Errorf("CreateEventW: %w", err)
	}
	return windows.Handle(r), nil
}

func closeHandle(h windows.Handle) {
	if h == 0 {
		return
	}
	procCloseHandle.Call(uintptr(h))
}
