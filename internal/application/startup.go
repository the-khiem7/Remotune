//go:build windows

package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/khiemnguyen/remotune/internal/crd"
)

const pollInterval = 2000 // milliseconds
var reconciliationInterval = 30 * time.Second

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

type Bootstrapper interface {
	Bootstrap() (crd.BootstrapResult, error)
	SubscribeAfterBookmark(bookmark string) (Subscription, error)
}
type Subscription interface {
	Poll(max int, timeoutMs uint32) ([]crd.Transition, int, error)
	Close()
}
type LiveDetector struct{}

func (LiveDetector) Bootstrap() (crd.BootstrapResult, error) { return crd.Bootstrap() }

func (LiveDetector) SubscribeAfterBookmark(bookmark string) (Subscription, error) {
	return crd.SubscribeAfterBookmark(bookmark)
}
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
func activeAsHistory(s crd.Snapshot) []crd.Transition {
	out := make([]crd.Transition, 0, len(s.ActiveSessions))
	for _, sess := range s.ActiveSessions {
		out = append(out, sess.ConnectedAt)
	}
	return out
}
