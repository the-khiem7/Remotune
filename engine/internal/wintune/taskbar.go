//go:build windows

package wintune

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	stuckRectsKey  = `Software\Microsoft\Windows\CurrentVersion\Explorer\StuckRects3`
	stuckRectsName = "Settings"
	// stuckRectsAutoHideByte is the offset of the flags byte inside the Settings blob.
	stuckRectsAutoHideByte = 8
	// stuckRectsAutoHideBit is the auto-hide flag within that byte.
	stuckRectsAutoHideBit = 0x01
)

// settleDelay is how long the shell needs before a taskbar change is observable.
// Measured at roughly 1.2 s during Phase 0; kept slightly higher for headroom.
var settleDelay = 1500 * time.Millisecond

// TaskbarManager reads and writes taskbar auto-hide.
//
// It writes BOTH layers on every change:
//
//	ABM_SETSTATE   immediate live effect
//	StuckRects3    what Explorer persists and can reconcile back from
//
// Writing only the API layer is not durable. The live state and the persisted value
// were observed to diverge, and an override applied only through the API later
// reverted on its own. Writing both removes the divergence window.
type TaskbarManager struct{}

// TaskbarState is the observed auto-hide state across both layers.
type TaskbarState struct {
	// ABMState is the raw appbar state word, retained so unrelated bits can be preserved.
	ABMState uint32
	// Live is auto-hide as the shell currently behaves.
	Live bool
	// Persisted is auto-hide as Explorer has it stored.
	Persisted bool
}

// Agreed reports whether the two layers match. Disagreement is a health signal: it
// means the override can be silently reverted at any time.
func (s TaskbarState) Agreed() bool { return s.Live == s.Persisted }

// GetState reads both layers.
func (m *TaskbarManager) GetState() (TaskbarState, error) {
	abm := appBarGetState()
	persisted, err := readPersistedAutoHide()
	if err != nil {
		return TaskbarState{}, err
	}
	return TaskbarState{
		ABMState:  abm,
		Live:      abm&ABSAutoHide != 0,
		Persisted: persisted,
	}, nil
}

// GetAutoHide returns the live auto-hide value.
func (m *TaskbarManager) GetAutoHide() (bool, error) {
	s, err := m.GetState()
	if err != nil {
		return false, err
	}
	return s.Live, nil
}

// Snapshot captures the current auto-hide state for later exact restoration.
func (m *TaskbarManager) Snapshot() (*TaskbarSnapshot, error) {
	s, err := m.GetState()
	if err != nil {
		return nil, err
	}
	// The live value is what the user experiences, so it is the baseline to restore.
	return &TaskbarSnapshot{
		AutoHide:            s.Live,
		ABMState:            s.ABMState,
		LivePersistedAgreed: s.Agreed(),
	}, nil
}

// SetAutoHide writes the desired auto-hide value to both layers, changing only the
// ABS_AUTOHIDE bit and only bit 0 of the persisted flags byte, then verifies the
// observable outcome.
func (m *TaskbarManager) SetAutoHide(on bool) (CategoryResult, error) {
	res := CategoryResult{Category: CategoryTaskbar}

	before, err := m.GetState()
	if err != nil {
		res.Err = err
		return res, err
	}

	if before.Live == on && before.Persisted == on {
		// Already correct on both layers. Idempotent no-op, still reported as verified.
		res.Verified = true
		return res, nil
	}

	// live layer: flip only ABS_AUTOHIDE, carry every other bit through
	target := before.ABMState
	if on {
		target |= ABSAutoHide
	} else {
		target &^= ABSAutoHide
	}
	// ABM_SETSTATE returns FALSE on some builds even when it works, so the error is
	// recorded but the observable outcome decides success.
	setErr := appBarSetState(target)

	// persisted layer: flip only bit 0 of the flags byte
	if err := writePersistedAutoHide(on); err != nil {
		res.Err = fmt.Errorf("persist auto-hide: %w", err)
		return res, res.Err
	}

	time.Sleep(settleDelay)

	after, err := m.GetState()
	if err != nil {
		res.Err = err
		return res, err
	}
	res.Changed = after.Live != before.Live || after.Persisted != before.Persisted

	if after.Live != on {
		res.Err = fmt.Errorf("live auto-hide is %v, want %v (ABM_SETSTATE reported: %v)", after.Live, on, setErr)
		return res, res.Err
	}
	if after.Persisted != on {
		res.Err = fmt.Errorf("persisted auto-hide is %v, want %v", after.Persisted, on)
		return res, res.Err
	}

	// Independent confirmation from an observable side effect rather than a re-read of
	// the same value we just wrote.
	if ok, err := verifyAutoHideByWorkArea(on); err == nil && !ok {
		res.Err = fmt.Errorf("work area does not reflect auto-hide=%v", on)
		return res, res.Err
	}

	res.Verified = true
	return res, nil
}

// Restore returns auto-hide to a captured baseline. A nil snapshot is not guessed at.
func (m *TaskbarManager) Restore(s *TaskbarSnapshot) (CategoryResult, error) {
	if s == nil {
		err := errors.New("no taskbar snapshot to restore")
		return CategoryResult{Category: CategoryTaskbar, Err: err}, err
	}
	return m.SetAutoHide(s.AutoHide)
}

func readPersistedAutoHide() (bool, error) {
	b, err := readStuckRects()
	if err != nil {
		return false, err
	}
	return b[stuckRectsAutoHideByte]&stuckRectsAutoHideBit != 0, nil
}

func readStuckRects() ([]byte, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, stuckRectsKey, registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", stuckRectsKey, err)
	}
	defer k.Close()

	b, _, err := k.GetBinaryValue(stuckRectsName)
	if err != nil {
		return nil, fmt.Errorf("read %s\\%s: %w", stuckRectsKey, stuckRectsName, err)
	}
	if len(b) <= stuckRectsAutoHideByte {
		return nil, fmt.Errorf("%s blob is %d bytes, need more than %d", stuckRectsName, len(b), stuckRectsAutoHideByte)
	}
	return b, nil
}

// writePersistedAutoHide flips only the auto-hide bit inside the current blob, so
// unrelated taskbar settings such as edge or size are never reverted.
func writePersistedAutoHide(on bool) error {
	b, err := readStuckRects()
	if err != nil {
		return err
	}
	if on {
		b[stuckRectsAutoHideByte] |= stuckRectsAutoHideBit
	} else {
		b[stuckRectsAutoHideByte] &^= stuckRectsAutoHideBit
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, stuckRectsKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open %s for write: %w", stuckRectsKey, err)
	}
	defer k.Close()
	return k.SetBinaryValue(stuckRectsName, b)
}

// verifyAutoHideByWorkArea confirms auto-hide from an observable side effect: an
// auto-hiding taskbar reserves no work area, so the work area fills the screen.
//
// Reported as (ok, err). A nil error with ok=false is a genuine mismatch.
func verifyAutoHideByWorkArea(expectAutoHide bool) (bool, error) {
	wa, err := primaryWorkArea()
	if err != nil {
		return false, err
	}
	screenW := getSystemMetrics(smCXScreen)
	screenH := getSystemMetrics(smCYScreen)
	if screenW == 0 || screenH == 0 {
		return false, errors.New("could not read screen metrics")
	}
	full := (wa.Right-wa.Left) == screenW && (wa.Bottom-wa.Top) == screenH
	return full == expectAutoHide, nil
}
