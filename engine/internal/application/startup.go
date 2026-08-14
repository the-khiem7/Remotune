//go:build windows

package application

import (
	"context"
	"fmt"

	"github.com/khiemnguyen/remotune/engine/internal/crd"
)

// pollInterval is how often Run polls the live subscription for new transitions.
// crd.Subscription.Poll blocks internally up to this timeout waiting on the signal
// event, so this is not busy polling; it bounds how promptly Quit/context
// cancellation is noticed between events.
const pollInterval = 2000 // milliseconds

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
func Run(ctx context.Context, c *Coordinator, det Bootstrapper) error {
	boot, err := det.Bootstrap()
	if err != nil {
		return fmt.Errorf("startup bootstrap: %w", err)
	}
	if err := c.Bootstrap(boot.Snapshot.State); err != nil {
		return fmt.Errorf("startup reconcile: %w", err)
	}

	sub, err := det.SubscribeAfterBookmark(boot.Bookmark)
	if err != nil {
		return fmt.Errorf("startup subscribe: %w", err)
	}
	defer sub.Close()

	// Transitions since the bootstrap snapshot was taken must be folded in before
	// entering the poll loop, using the SAME reconstruction the detector already
	// proved (Reconstruct), seeded with the bootstrap's own active-session state
	// rather than an empty one, so a transition delivered here does not appear to
	// be starting from a falsely-disconnected state.
	observed := boot.Snapshot

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		transitions, _, err := sub.Poll(32, pollInterval)
		if err != nil {
			// A poll error is a detector health problem, not a coordinator state
			// change; Remotune keeps whatever CRD state it last observed rather than
			// guessing, and diagnostics surfaces the error (belongs with Phase 4/5
			// diagnostics wiring; here it simply propagates to the caller's log).
			continue
		}
		if len(transitions) == 0 {
			continue
		}

		merged := append(activeAsHistory(observed), transitions...)
		observed = crd.Reconstruct(merged)

		if err := c.Observe(observed.State); err != nil {
			return fmt.Errorf("observe: %w", err)
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
