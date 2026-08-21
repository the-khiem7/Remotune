//go:build windows

package wintune

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	stuckRectsKey          = `Software\Microsoft\Windows\CurrentVersion\Explorer\StuckRects3`
	stuckRectsName         = "Settings"
	stuckRectsAutoHideByte = 8
	stuckRectsAutoHideBit  = 0x01
)

var settleDelay = 1500 * time.Millisecond

type TaskbarManager struct{}
type TaskbarState struct {
	ABMState  uint32
	Live      bool
	Persisted bool
}

func (s TaskbarState) Agreed() bool { return s.Live == s.Persisted }
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
func (m *TaskbarManager) GetAutoHide() (bool, error) {
	s, err := m.GetState()
	if err != nil {
		return false, err
	}
	return s.Live, nil
}
func (m *TaskbarManager) Snapshot() (*TaskbarSnapshot, error) {
	s, err := m.GetState()
	if err != nil {
		return nil, err
	}
	return &TaskbarSnapshot{
		AutoHide:            s.Live,
		ABMState:            s.ABMState,
		LivePersistedAgreed: s.Agreed(),
	}, nil
}
func (m *TaskbarManager) SetAutoHide(on bool) (CategoryResult, error) {
	res := CategoryResult{Category: CategoryTaskbar}

	before, err := m.GetState()
	if err != nil {
		res.Err = err
		return res, err
	}

	if before.Live == on && before.Persisted == on {
		res.Verified = true
		return res, nil
	}
	target := before.ABMState
	if on {
		target |= ABSAutoHide
	} else {
		target &^= ABSAutoHide
	}
	setErr := appBarSetState(target)
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
	if ok, err := verifyAutoHideByWorkArea(on); err == nil && !ok {
		res.Err = fmt.Errorf("work area does not reflect auto-hide=%v", on)
		return res, res.Err
	}

	res.Verified = true
	return res, nil
}
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
