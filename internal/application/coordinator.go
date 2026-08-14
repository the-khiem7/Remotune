//go:build windows

package application

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/khiemnguyen/remotune/internal/crd"
	"github.com/khiemnguyen/remotune/internal/wintune"
)

// VisualEffectsAdapter is the subset of *wintune.VisualEffectsManager the coordinator
// needs. Declaring it locally, rather than depending on the concrete type, is what
// lets tests exercise the coordinator's transition logic with a fake instead of
// mutating the operator's real desktop on every run — the same lesson Phase 1 learned
// the hard way about needing fast, repeatable, non-mutating test cycles.
type VisualEffectsAdapter interface {
	Snapshot() (*wintune.VisualEffectsSnapshot, error)
	ApplyBestPerformance() (wintune.CategoryResult, error)
	Restore(*wintune.VisualEffectsSnapshot) (wintune.CategoryResult, error)
}

// TaskbarAdapter is the subset of *wintune.TaskbarManager the coordinator needs.
type TaskbarAdapter interface {
	Snapshot() (*wintune.TaskbarSnapshot, error)
	SetAutoHide(bool) (wintune.CategoryResult, error)
	Restore(*wintune.TaskbarSnapshot) (wintune.CategoryResult, error)
}

// AutomationConfig is the user-controlled part of desired-state derivation:
// observed CRD state + automation enabled/paused + enabled categories + persisted
// ownership = desired Windows state (docs/baseline/remotune.sourcecode.md,
// "StateCoordinator"). Enabled/Paused and the per-category flags are config; CRD
// state and ownership are runtime facts tracked separately by the coordinator.
type AutomationConfig struct {
	Enabled       bool
	VisualEffects bool
	Taskbar       bool
}

// anyCategoryEnabled reports whether there is anything for an apply to do at all.
func (c AutomationConfig) anyCategoryEnabled() bool {
	return c.VisualEffects || c.Taskbar
}

// ErrShuttingDown is returned by transition methods called after Quit has begun.
var ErrShuttingDown = errors.New("coordinator is shutting down")

// Coordinator is the single owner of Windows state transitions
// (docs/baseline/remotune.sourcecode.md, "One coordinator owns all state
// transitions"). All exported methods serialize through mu, so there are never
// concurrent Visual Effects or taskbar writes; if an observation arrives while a
// transition is in flight, it blocks until that transition finishes and then
// reconciles against the now-current state, which is how "the latest desired state
// eventually wins" is achieved without a separate queue.
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

// NewCoordinator constructs a coordinator with no observed CRD state and no owned
// snapshot. Call Bootstrap once before any other method, mirroring the mandatory
// startup order in docs/baseline/remotune.sourcecode.md ("Startup flow").
func NewCoordinator(store *RecoveryStore, ve VisualEffectsAdapter, tb TaskbarAdapter, cfg AutomationConfig) *Coordinator {
	return &Coordinator{
		store: store,
		ve:    ve,
		tb:    tb,
		cfg:   cfg,
		state: TuningUnknown,
	}
}

// Status is the read-only snapshot of coordinator state for diagnostics and UI. It
// never mutates anything and is safe to poll.
type Status struct {
	Tuning            TuningState
	CRD               crd.State
	AutomationEnabled bool
	Paused            bool
	Owned             bool
}

// Status returns the current coordinator status.
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

// Bootstrap performs the startup reconciliation step
// (docs/baseline/remotune.sourcecode.md, "Startup flow" steps 3-8, condensed to the
// parts this coordinator owns): load persisted recovery state, take the detector's
// already-reconstructed CRD state, and reconcile.
//
// Recovery examples this implements, verbatim from the baseline:
//   - persisted ownership + still connected  -> reconcile without replacing baseline
//     (the same idempotent re-apply path a normal repeated-connect uses)
//   - persisted ownership + disconnected     -> restore the snapshot
//   - no ownership record                    -> Baseline, regardless of how Windows
//     currently looks; a tuned-looking system with no recovery file is never assumed
//     to be Remotune's doing (ledger decision 13)
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
		// A store I/O error (not an invalid/corrupt file, which Load already folds
		// into ErrNoRecovery) is not the same as "definitely no recovery." Surface it
		// as Recovery Required rather than silently proceeding as if clean.
		c.state = TuningRecoveryRequired
		return err
	}

	return c.reconcileLocked()
}

// Observe reports the detector's current CRD state, from either the real-time
// subscription or a reconciliation replay, and reconciles.
func (c *Coordinator) Observe(state crd.State) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quitting {
		return ErrShuttingDown
	}
	c.crdState = state
	return c.reconcileLocked()
}

// Pause restores any owned state, then stops automatic reactions until Resume.
// (docs/baseline/remotune.sourcecode.md, "Pause Automation"). It always sets Paused,
// even if the restore attempt itself reports Partial/Error, because pausing is a
// deliberate operator command to stop automation, and the failure remains separately
// visible through Status().Tuning.
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

// Resume re-enables automatic reconciliation and immediately reconciles against the
// current CRD state, applying enabled categories again if warranted.
func (c *Coordinator) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.quitting {
		return ErrShuttingDown
	}
	c.paused = false
	return c.reconcileLocked()
}

// RestoreNow acts only on a valid, owned snapshot and otherwise reports
// ErrNoRecovery without guessing (docs/baseline/remotune.sourcecode.md, "Restore
// Now"). Unlike automatic reconciliation, this is an explicit operator command: it
// restores regardless of the currently observed CRD state.
func (c *Coordinator) RestoreNow() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owned == nil {
		return ErrNoRecovery
	}
	return c.restoreLocked()
}

// Quit stops accepting new automatic transitions, restores any owned state, and
// leaves the store in its final, honest condition (docs/baseline/remotune.sourcecode.md,
// "Explicit Quit"). It is idempotent: calling it again is a no-op returning nil.
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

// desiredOwned derives whether Windows should currently be in the owned/tuned state,
// per the desired-state formula in docs/baseline/remotune.sourcecode.md.
func (c *Coordinator) desiredOwned() bool {
	return c.cfg.Enabled &&
		!c.paused &&
		c.crdState == crd.StateConnected &&
		c.cfg.anyCategoryEnabled()
}

// reconcileLocked derives desired state from current fields and drives one apply or
// restore transaction if warranted. Callers must hold mu.
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

// applyLocked implements the Apply transaction flow
// (docs/baseline/remotune.sourcecode.md, "Transaction flows" / "Apply"). If an owned
// snapshot already exists (a repeated apply, or a bootstrap reconciliation with
// persisted ownership), the original baseline is never replaced; only the apply
// actions and verification run again. Callers must hold mu.
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
		// Durable BEFORE any mutation: a crash between here and the writes below must
		// still leave a valid path back to this exact baseline.
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
	// Partial failure (or total failure) must remain visible and retryable; the
	// recovery data written above is deliberately NOT retired here.
	c.state = TuningPartialError
	return result.Err()
}

// restoreLocked implements the Restore transaction flow
// (docs/baseline/remotune.sourcecode.md, "Transaction flows" / "Restore"). Callers
// must hold mu and must have already confirmed c.owned != nil.
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
		// Ownership is cleared and the snapshot retired ONLY after a verified
		// success; a partial restore must retain the recovery data for retry.
		if err := c.store.Retire(); err != nil {
			// The Windows state was verified restored, but the durable record of
			// that could not be cleaned up. This is not a tuning failure: report it
			// distinctly rather than as Partial/Error, which would incorrectly imply
			// Windows itself is left in an inconsistent state.
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
