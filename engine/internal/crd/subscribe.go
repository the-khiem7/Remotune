//go:build windows

package crd

import (
	"fmt"
)

// Subscription is a live, pull-mode Event Log subscription. It must be closed with
// Close when no longer needed.
//
// This is Phase 2 Group B: it depends on the underlying EvtSubscribe wiring being
// correct, which was proven manually in the Phase 0 spike (two independent probes,
// reader- and watcher-based, each replayed 5 of 5 expected records from a bookmark).
// It is wired here but deliberately NOT exercised against a live CRD session by an
// automated test in this change, because doing so requires the operator to disconnect
// their current, real CRD session. See docs/baseline/remotune.roadmap.md, Phase 2,
// for the observation protocol to use when that becomes possible.
type Subscription struct {
	handle evtHandle
}

// SubscribeAfterBookmark starts a subscription positioned after the given bookmark XML
// (as returned by HistoryResult.Bookmark), with read-existing-events behavior enabled.
//
// This is the gap-free handover: any transition that occurred between the historical
// query that produced the bookmark and this call is replayed rather than lost.
// Disabling this behavior silently degrades to future-only delivery (ledger decision
// 37); there is deliberately no parameter to do that.
func SubscribeAfterBookmark(bookmarkXML string) (*Subscription, error) {
	if bookmarkXML == "" {
		return nil, fmt.Errorf("SubscribeAfterBookmark: empty bookmark; use QueryHistory first")
	}
	bm, err := evtCreateBookmark(bookmarkXML)
	if err != nil {
		return nil, fmt.Errorf("recreate bookmark: %w", err)
	}
	defer evtClose(bm)

	h, err := evtSubscribe(Channel, XPath, bm, evtSubscribeStartAfterBookmark)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	return &Subscription{handle: h}, nil
}

// Close releases the subscription's handle. Safe to call once; the underlying handle
// is not reused afterward.
func (s *Subscription) Close() {
	if s == nil {
		return
	}
	evtClose(s.handle)
	s.handle = 0
}

// Poll pulls up to max pending transitions without blocking longer than timeoutMs.
// Events that fail to parse are skipped and counted, mirroring QueryHistory, rather
// than aborting the whole poll.
func (s *Subscription) Poll(max int, timeoutMs uint32) ([]Transition, int, error) {
	if s == nil || s.handle == 0 {
		return nil, 0, fmt.Errorf("poll on closed subscription")
	}
	if max <= 0 {
		max = batchSize
	}
	handles := make([]evtHandle, max)
	batch, err := evtNext(s.handle, handles, timeoutMs)
	if err != nil {
		return nil, 0, fmt.Errorf("poll subscription: %w", err)
	}

	var out []Transition
	var skipped int
	for _, h := range batch {
		xmlStr, rerr := evtRenderEventXML(h)
		evtClose(h)
		if rerr != nil {
			skipped++
			continue
		}
		t, perr := ParseTransition(xmlStr)
		if perr != nil {
			skipped++
			continue
		}
		out = append(out, t)
	}
	return out, skipped, nil
}
