//go:build windows

package application

import (
	"context"
	"fmt"

	"github.com/khiemnguyen/remotune/engine/internal/crd"
)

const pollInterval = 2000 // milliseconds
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
	observed := boot.Snapshot

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		transitions, _, err := sub.Poll(32, pollInterval)
		if err != nil {
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
func activeAsHistory(s crd.Snapshot) []crd.Transition {
	out := make([]crd.Transition, 0, len(s.ActiveSessions))
	for _, sess := range s.ActiveSessions {
		out = append(out, sess.ConnectedAt)
	}
	return out
}
