//go:build windows

package crd

import (
	"fmt"

	"golang.org/x/sys/windows"
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
	handle      evtHandle
	signalEvent windows.Handle
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

	sig, err := createManualResetEvent()
	if err != nil {
		return nil, fmt.Errorf("create signal event: %w", err)
	}

	h, err := evtSubscribe(Channel, XPath, bm, sig, evtSubscribeStartAfterBookmark)
	if err != nil {
		closeHandle(sig)
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	return &Subscription{handle: h, signalEvent: sig}, nil
}

// Close releases the subscription's handles. Safe to call once; the underlying handles
// are not reused afterward.
func (s *Subscription) Close() {
	if s == nil {
		return
	}
	evtClose(s.handle)
	s.handle = 0
	closeHandle(s.signalEvent)
	s.signalEvent = 0
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

	// Pull-mode contract: wait on the signal event, then call EvtNext with timeout 0.
	// If nothing signaled within timeoutMs, there is nothing new; return empty rather
	// than calling EvtNext at all.
	if !waitForSignal(s.signalEvent, timeoutMs) {
		return nil, 0, nil
	}

	handles := make([]evtHandle, max)
	batch, err := evtNext(s.handle, handles, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("poll subscription: %w", err)
	}
	if len(batch) < max {
		// The result set is drained for now (EvtNext returned fewer than requested,
		// or hit ERROR_NO_MORE_ITEMS internally). The event is manual-reset and stays
		// signaled until cleared, so it is reset here to make the next call's wait
		// meaningful instead of always returning immediately.
		resetSignal(s.signalEvent)
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
