//go:build windows

package crd

import "fmt"

type BootstrapResult struct {
	Snapshot         Snapshot
	Bookmark         string
	SkippedMalformed int
}

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
