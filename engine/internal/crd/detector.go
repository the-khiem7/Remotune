//go:build windows

package crd

import "fmt"

// BootstrapResult is the outcome of the mandatory startup sequence: historical replay
// followed by state reconstruction, with the bookmark needed to continue into a live
// subscription without a gap (Phase 2 Group B).
type BootstrapResult struct {
	Snapshot         Snapshot
	Bookmark         string
	SkippedMalformed int
}

// Bootstrap performs the startup order required by the product baseline
// (docs/baseline/remotune.sourcecode.md, "Startup detection order is mandatory"):
// query and reconstruct current state before any live subscription is established.
//
// It is read-only and does not touch any live CRD session.
func Bootstrap() (BootstrapResult, error) {
	hist, err := QueryHistory()
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("bootstrap: %w", err)
	}
	return BootstrapResult{
		Snapshot:         Reconstruct(hist.Transitions),
		Bookmark:         hist.Bookmark,
		SkippedMalformed: hist.SkippedMalformed,
	}, nil
}
