//go:build windows

package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/khiemnguyen/remotune/internal/crd"
)

// pollInterval is how often Run polls the live subscription for new transitions.
// crd.Subscription.Poll blocks internally up to this timeout waiting on the signal
// event, so this is not busy polling; it bounds how promptly Quit/context
// cancellation is noticed between events.
const pollInterval = 2000 // milliseconds

// reconciliationInterval bounds how long the UI can remain stale if the Event Log
// subscription misses a signal without reporting an error. Reconciliation queries
// the same redacted history as startup; it never guesses a connection state.
var reconciliationInterval = 30 * time.Second

// DetectorStatus is the operator-facing, privacy-safe health snapshot of the CRD
// detector. It intentionally excludes session IDs and EventData, which can contain
// account information. It is embedded in application.Status for the Vue and tray
// surfaces.
type DetectorStatus struct {
	Health                string
	LastTransition        string
	LastTransitionAt      time.Time
	LastRecordID          uint64
	LastPollError         string
	ConsecutivePollErrors int
	SkippedRecords        int
	LastReconciledAt      time.Time
}

// DetectorDiagnostics serializes detector health updates from the run loop while
// Status is being polled by the UI and tray.
type DetectorDiagnostics struct {
	mu     sync.RWMutex
	status DetectorStatus
}

func NewDetectorDiagnostics() *DetectorDiagnostics {
	return &DetectorDiagnostics{status: DetectorStatus{Health: "Starting"}}
}

func (d *DetectorDiagnostics) Status() DetectorStatus {
	if d == nil {
		return DetectorStatus{Health: "Unavailable"}
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.status
}

func (d *DetectorDiagnostics) recordBootstrap(snapshot crd.Snapshot) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status.Health = "Healthy"
	d.status.LastPollError = ""
	d.status.ConsecutivePollErrors = 0
	d.status.LastReconciledAt = time.Now()
	if snapshot.HasLastTransition {
		d.recordTransitionLocked(snapshot.LastTransition)
	}
}

func (d *DetectorDiagnostics) recordTransition(t crd.Transition) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.recordTransitionLocked(t)
}

func (d *DetectorDiagnostics) recordTransitionLocked(t crd.Transition) {
	if t.RecordID >= d.status.LastRecordID {
		d.status.LastTransition = t.Kind.String()
		d.status.LastTransitionAt = t.Time
		d.status.LastRecordID = t.RecordID
	}
}

func (d *DetectorDiagnostics) recordPoll(skipped int, err error) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status.SkippedRecords += skipped
	if err != nil {
		d.status.Health = "Degraded"
		d.status.LastPollError = err.Error()
		d.status.ConsecutivePollErrors++
		return
	}
	d.status.Health = "Healthy"
	d.status.LastPollError = ""
	d.status.ConsecutivePollErrors = 0
}

func (d *DetectorDiagnostics) recordFailure(err error) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status.Health = "Degraded"
	d.status.LastPollError = err.Error()
}

// Bootstrapper is the subset of the crd package's startup sequence the coordinator
// depends on, declared locally so it can be faked in tests instead of requiring a
// real Windows Event Log for every coordinator test (mirrors VisualEffectsAdapter /
// TaskbarAdapter above).
type Bootstrapper interface {
	Bootstrap() (crd.BootstrapResult, error)
	SubscribeAfterBookmark(bookmark string) (Subscription, error)
}

// Subscription is the subset of *crd.Subscription the run loop depends on.
type Subscription interface {
	Poll(max int, timeoutMs uint32) ([]crd.Transition, int, error)
	Close()
}

// LiveDetector adapts the real crd package functions to the Bootstrapper interface.
// This indirection exists only so Run's tests can substitute a fake without needing
// a real Event Log or a real CRD session (Phase 2 already proved the real thing
// works; Phase 3's own tests are about coordinator logic, not re-proving Phase 2).
type LiveDetector struct{}

func (LiveDetector) Bootstrap() (crd.BootstrapResult, error) { return crd.Bootstrap() }

func (LiveDetector) SubscribeAfterBookmark(bookmark string) (Subscription, error) {
	return crd.SubscribeAfterBookmark(bookmark)
}

// Run performs the mandatory startup sequence and then drives the coordinator from
// live CRD transitions until ctx is canceled or Quit is called.
//
// Startup order (docs/baseline/remotune.sourcecode.md, "Startup flow", the parts this
// function owns): query and reconstruct current CRD state, reconcile the coordinator
// against it, THEN establish the real-time subscription seeded from the bootstrap's
// bookmark. This ordering, and specifically using the bootstrap's own bookmark rather
// than a fresh one, is what closes the query/subscription race (ledger decision 37):
// no transition between the historical read and the live subscription can be lost.
func Run(ctx context.Context, c *Coordinator, det Bootstrapper, diagnostics ...*DetectorDiagnostics) error {
	var diag *DetectorDiagnostics
	if len(diagnostics) > 0 {
		diag = diagnostics[0]
	}
	boot, err := det.Bootstrap()
	if err != nil {
		diag.recordFailure(fmt.Errorf("startup bootstrap: %w", err))
		return fmt.Errorf("startup bootstrap: %w", err)
	}
	diag.recordBootstrap(boot.Snapshot)
	if err := c.Bootstrap(boot.Snapshot.State); err != nil {
		diag.recordFailure(fmt.Errorf("startup reconcile: %w", err))
		return fmt.Errorf("startup reconcile: %w", err)
	}

	sub, err := det.SubscribeAfterBookmark(boot.Bookmark)
	if err != nil {
		diag.recordFailure(fmt.Errorf("startup subscribe: %w", err))
		return fmt.Errorf("startup subscribe: %w", err)
	}
	defer sub.Close()

	// Transitions since the bootstrap snapshot was taken must be folded in before
	// entering the poll loop, using the SAME reconstruction the detector already
	// proved (Reconstruct), seeded with the bootstrap's own active-session state
	// rather than an empty one, so a transition delivered here does not appear to
	// be starting from a falsely-disconnected state.
	observed := boot.Snapshot
	nextReconciliation := time.Now().Add(reconciliationInterval)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		transitions, skipped, err := sub.Poll(32, pollInterval)
		diag.recordPoll(skipped, err)
		if err != nil {
			// Keep the last observed state, then use the bounded history replay below
			// to recover from a missed subscription signal without guessing.
		} else if len(transitions) > 0 {
			merged := append(activeAsHistory(observed), transitions...)
			observed = crd.Reconstruct(merged)
			for _, transition := range transitions {
				diag.recordTransition(transition)
			}

			if err := c.Observe(observed.State); err != nil {
				diag.recordFailure(fmt.Errorf("observe: %w", err))
				return fmt.Errorf("observe: %w", err)
			}
		}

		if time.Now().Before(nextReconciliation) {
			continue
		}
		nextReconciliation = time.Now().Add(reconciliationInterval)
		boot, err := det.Bootstrap()
		if err != nil {
			diag.recordFailure(fmt.Errorf("reconcile bootstrap: %w", err))
			continue
		}
		observed = boot.Snapshot
		diag.recordBootstrap(observed)
		if err := c.Observe(observed.State); err != nil {
			diag.recordFailure(fmt.Errorf("reconcile observe: %w", err))
			return fmt.Errorf("reconcile observe: %w", err)
		}
	}
}

// activeAsHistory turns a reconstructed snapshot's active sessions back into
// synthetic "still connected" transition inputs, so Reconstruct can be re-run
// incrementally: feeding it the previously active connects plus newly polled
// transitions reproduces the same result as feeding it the full history, without
// this process needing to retain the full history itself.
func activeAsHistory(s crd.Snapshot) []crd.Transition {
	out := make([]crd.Transition, 0, len(s.ActiveSessions))
	for _, sess := range s.ActiveSessions {
		out = append(out, sess.ConnectedAt)
	}
	return out
}
