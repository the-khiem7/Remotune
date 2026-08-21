//go:build windows

package application

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/khiemnguyen/remotune/engine/internal/crd"
	"github.com/khiemnguyen/remotune/engine/internal/wintune"
)

type VisualEffectsAdapter interface {
	Snapshot() (*wintune.VisualEffectsSnapshot, error)
	ApplyBestPerformance() (wintune.CategoryResult, error)
	Restore(*wintune.VisualEffectsSnapshot) (wintune.CategoryResult, error)
}
type TaskbarAdapter interface {
	Snapshot() (*wintune.TaskbarSnapshot, error)
	SetAutoHide(bool) (wintune.CategoryResult, error)
	Restore(*wintune.TaskbarSnapshot) (wintune.CategoryResult, error)
}
type AutomationConfig struct {
	Enabled       bool
	VisualEffects bool
	Taskbar       bool
}

func (c AutomationConfig) anyCategoryEnabled() bool {
	return c.VisualEffects || c.Taskbar
}

var ErrShuttingDown = errors.New("coordinator is shutting down")

type Coordinator struct {
	mu sync.Mutex

	store *RecoveryStore
	ve    VisualEffectsAdapter
	tb    TaskbarAdapter

	cfg      AutomationConfig
	crdState crd.State
	paused   bool
	quitting bool

	state TuningState
	owned *wintune.Snapshot
}

func NewCoordinator(store *RecoveryStore, ve VisualEffectsAdapter, tb TaskbarAdapter, cfg AutomationConfig) *Coordinator {
	return &Coordinator{
		store: store,
		ve:    ve,
		tb:    tb,
		cfg:   cfg,
		state: TuningUnknown,
	}
}

type Status struct {
	Tuning            TuningState
	CRD               crd.State
	AutomationEnabled bool
	Paused            bool
	Owned             bool
}

func (c *Coordinator) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Status{
		Tuning:            c.state,
		CRD:               c.crdState,
		AutomationEnabled: c.cfg.Enabled,
		Paused:            c.paused,
		Owned:             c.owned != nil,
	}
}
func (c *Coordinator) Bootstrap(observedCRD crd.State) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.crdState = observedCRD

	snap, err := c.store.Load()
	switch {
	case err == nil:
		c.owned = snap
	case errors.Is(err, ErrNoRecovery):
		c.owned = nil
	default:
		c.state = TuningRecoveryRequired
		return err
	}

	return c.reconcileLocked()
}
func (c *Coordinator) Observe(state crd.State) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quitting {
		return ErrShuttingDown
	}
	c.crdState = state
	return c.reconcileLocked()
}
func (c *Coordinator) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quitting {
		return ErrShuttingDown
	}
	var err error
	if c.owned != nil {
		err = c.restoreLocked()
	}
	c.paused = true
	return err
}
func (c *Coordinator) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quitting {
		return ErrShuttingDown
	}
	c.paused = false
	return c.reconcileLocked()
}
func (c *Coordinator) RestoreNow() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owned == nil {
		return ErrNoRecovery
	}
	return c.restoreLocked()
}
func (c *Coordinator) Quit() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quitting {
		return nil
	}
	c.quitting = true
	if c.owned != nil {
		return c.restoreLocked()
	}
	return nil
}
func (c *Coordinator) desiredOwned() bool {
	return c.cfg.Enabled &&
		!c.paused &&
		c.crdState == crd.StateConnected &&
		c.cfg.anyCategoryEnabled()
}
func (c *Coordinator) reconcileLocked() error {
	switch {
	case c.desiredOwned():
		return c.applyLocked()
	case c.owned != nil:
		return c.restoreLocked()
	default:
		if !c.state.IsTransient() {
			c.state = TuningBaseline
		}
		return nil
	}
}
func (c *Coordinator) applyLocked() error {
	if c.owned == nil {
		snap := &wintune.Snapshot{
			SchemaVersion: wintune.SnapshotSchemaVersion,
			CapturedAt:    time.Now(),
			Machine:       hostname(),
		}
		if c.cfg.VisualEffects {
			ve, err := c.ve.Snapshot()
			if err != nil {
				return err
			}
			snap.VisualEffects = ve
		}
		if c.cfg.Taskbar {
			tb, err := c.tb.Snapshot()
			if err != nil {
				return err
			}
			snap.Taskbar = tb
		}
		if err := c.store.Save(snap); err != nil {
			return err
		}
		c.owned = snap
	}

	c.state = TuningApplying

	var results []wintune.CategoryResult
	if c.cfg.VisualEffects {
		res, _ := c.ve.ApplyBestPerformance()
		results = append(results, res)
	}
	if c.cfg.Taskbar {
		res, _ := c.tb.SetAutoHide(false)
		results = append(results, res)
	}

	result := wintune.Result{Categories: results}
	if result.FullyVerified() {
		c.state = TuningActive
		return nil
	}
	c.state = TuningPartialError
	return result.Err()
}
func (c *Coordinator) restoreLocked() error {
	if c.owned == nil {
		return ErrNoRecovery
	}

	c.state = TuningRestoring

	var results []wintune.CategoryResult
	if c.owned.VisualEffects != nil {
		res, _ := c.ve.Restore(c.owned.VisualEffects)
		results = append(results, res)
	}
	if c.owned.Taskbar != nil {
		res, _ := c.tb.Restore(c.owned.Taskbar)
		results = append(results, res)
	}

	result := wintune.Result{Categories: results}
	if result.FullyVerified() {
		if err := c.store.Retire(); err != nil {
			c.state = TuningRecoveryRequired
			return err
		}
		c.owned = nil
		c.state = TuningBaseline
		return nil
	}
	c.state = TuningPartialError
	return result.Err()
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
