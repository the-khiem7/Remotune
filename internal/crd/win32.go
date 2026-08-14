//go:build windows

// Package crd implements the CRD (Chrome Remote Desktop) connection detector.
//
// It reads the Windows Event Log only. It never mutates Windows state and, like
// internal/wintune, has no dependency on Wails so it can be tested headlessly and
// migrated into the Wails project in Phase 4 without redesign.
//
// Evidence base: Phase 0 spike, 2026-08-14, recorded in
// docs/baseline/remotune.roadmap.md#phase-0-recorded-evidence.
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

// waitTimeout signals from WaitForSingleObject.
const (
	waitObject0 = 0x00000000
)

// waitForSignal blocks until the event is signaled or timeoutMs elapses, returning
// true if signaled. EvtSubscribe's pull-mode contract is: wait on SignalEvent, then
// call EvtNext with timeout 0. Calling EvtNext with a nonzero timeout in pull mode was
// measured directly to fail with ERROR_INVALID_OPERATION.
func waitForSignal(h windows.Handle, timeoutMs uint32) bool {
	r, _, _ := procWaitForSingleObject.Call(uintptr(h), uintptr(timeoutMs))
	return r == waitObject0
}

// resetSignal clears a manual-reset event so a future signal can be waited on again.
// Needed because EvtSubscribe's SignalEvent is manual-reset: once set, it stays set
// until explicitly reset.
func resetSignal(h windows.Handle) {
	procResetEvent.Call(uintptr(h))
}

// EvtQuery flags (subset used by this package).
const (
	evtQueryChannelPath      = 0x1
	evtQueryReverseDirection = 0x200
)

// EvtRender flags.
const (
	evtRenderEventXml    = 1
	evtRenderBookmarkXml = 2
)

// EvtSubscribe flags.
const (
	evtSubscribeToFutureEvents      = 1
	evtSubscribeStartAtOldestRecord = 2
	// EvtSubscribeStartAfterBookmark is the mode that closes the Phase 0 startup race:
	// historical EvtQuery replay, then subscribe from the bookmark of the last event
	// consumed, so no transition can fall in the gap. See ledger decision 37.
	evtSubscribeStartAfterBookmark = 3
)

// errorNoMoreItems is ERROR_NO_MORE_ITEMS (259). It signals a normal, non-error end of
// a result set, not a real failure, and is not part of x/sys/windows' named constants.
const errorNoMoreItems = syscall.Errno(259)

// evtHandle is an opaque EVT_HANDLE. Every non-nil handle must be closed with EvtClose.
type evtHandle uintptr

func (h evtHandle) valid() bool { return h != 0 }

func evtClose(h evtHandle) {
	if h == 0 {
		return
	}
	procEvtClose.Call(uintptr(h))
}

// evtQuery opens a query against a channel. direction controls whether EvtNext walks
// oldest-to-newest (false) or newest-to-oldest (true, used for finding the last event).
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

// evtNext pulls up to len(out) event handles from a query or subscription result set.
// It returns the handles actually returned; ERROR_NO_MORE_ITEMS is not an error, it
// means the result set is exhausted and is reported by returning an empty slice.
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

// evtRenderEventXML renders one event as its full XML representation.
func evtRenderEventXML(event evtHandle) (string, error) {
	return evtRenderXML(event, evtRenderEventXml)
}

func evtRenderBookmarkXML(bookmark evtHandle) (string, error) {
	return evtRenderXML(bookmark, evtRenderBookmarkXml)
}

func evtRenderXML(handle evtHandle, flags uint32) (string, error) {
	var used, propCount uint32
	// First call with a nil buffer to discover the required size.
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

// evtCreateBookmark creates a fresh, unpositioned bookmark, or one restored from a
// previously rendered bookmark XML string.
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

// evtUpdateBookmark positions a bookmark at the given event.
func evtUpdateBookmark(bookmark, event evtHandle) error {
	r, _, err := procEvtUpdateBookmark.Call(uintptr(bookmark), uintptr(event))
	if r == 0 {
		return fmt.Errorf("EvtUpdateBookmark: %w", err)
	}
	return nil
}

// evtSubscribe starts a pull-mode subscription. Pull mode requires a valid manual-reset
// SignalEvent handle (EvtSubscribe rejects NULL there with ERROR_INVALID_PARAMETER,
// measured directly against this machine); the caller polls with evtNext regardless,
// the event is only there to satisfy the API contract. Passing a bookmark with mode
// evtSubscribeStartAfterBookmark is the gap-free handover verified in Phase 0.
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

// createManualResetEvent creates an unnamed manual-reset event, created ALREADY
// SIGNALED. Required as EvtSubscribe's SignalEvent parameter in pull mode.
//
// The initial state must be signaled, matching Microsoft's canonical pull-subscription
// sample (CreateEvent(NULL, TRUE, TRUE, NULL)): the first wait is meant to fall through
// immediately so the caller drains any backlog/replay past the bookmark via EvtNext
// before ever blocking. Creating it unsignaled was measured directly on this machine to
// make EvtSubscribe's own signal never fire, even with events already queued past the
// bookmark; WaitForSingleObject on the same handle otherwise worked correctly.
func createManualResetEvent() (windows.Handle, error) {
	r, _, err := procCreateEventW.Call(0, 1 /*manual reset*/, 1 /*initial state: signaled*/, 0)
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
