//go:build windows

package crd

import (
	"testing"
	"time"
)

// TestSubscribeReplaysFromMidHistoryBookmark isolates the EvtSubscribe binding from any
// dependency on a NEW live event. It seeds a bookmark somewhere in the middle of the
// existing, already-recorded history, subscribes from it, and expects the events
// AFTER that bookmark to replay immediately (this is exactly the mechanism Phase 0
// proved manually with PowerShell: EventLogWatcher(bookmark, readExisting=true)
// replayed 5/5 expected records). If this test fails to receive anything, the bug is
// in the Go binding, independent of whether a real new CRD event ever arrives.
func TestSubscribeReplaysFromMidHistoryBookmark(t *testing.T) {
	hist, err := QueryHistory()
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if len(hist.Transitions) < 6 {
		t.Skip("not enough history on this machine to pick a mid-point bookmark")
	}

	// Re-query to get real EVT_HANDLEs (QueryHistory already closed its handles), and
	// position a bookmark on the event 5 from the end, so the last 4 should replay.
	q, err := evtQuery(Channel, XPath, false)
	if err != nil {
		t.Fatalf("evtQuery: %v", err)
	}
	defer evtClose(q)

	handles := make([]evtHandle, batchSize)
	var all []evtHandle
	for {
		batch, err := evtNext(q, handles, 0)
		if err != nil {
			t.Fatalf("evtNext: %v", err)
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
	}
	defer func() {
		for _, h := range all {
			evtClose(h)
		}
	}()

	if len(all) < 6 {
		t.Skip("not enough live handles retrieved")
	}

	anchorIdx := len(all) - 5 // bookmark here; events at anchorIdx+1..end should replay
	bm, err := evtCreateBookmark("")
	if err != nil {
		t.Fatalf("evtCreateBookmark: %v", err)
	}
	defer evtClose(bm)
	if err := evtUpdateBookmark(bm, all[anchorIdx]); err != nil {
		t.Fatalf("evtUpdateBookmark: %v", err)
	}
	bmXML, err := evtRenderBookmarkXML(bm)
	if err != nil {
		t.Fatalf("evtRenderBookmarkXML: %v", err)
	}
	expectedReplay := len(all) - 1 - anchorIdx
	t.Logf("anchored bookmark at index %d of %d; expecting %d event(s) to replay", anchorIdx, len(all), expectedReplay)

	sub, err := SubscribeAfterBookmark(bmXML)
	if err != nil {
		t.Fatalf("SubscribeAfterBookmark: %v", err)
	}
	defer sub.Close()

	var got []Transition
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && len(got) < expectedReplay {
		ts, skipped, err := sub.Poll(32, 1000)
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		if skipped > 0 {
			t.Logf("skipped %d malformed event(s)", skipped)
		}
		got = append(got, ts...)
	}

	t.Logf("replayed %d transition(s): %v", len(got), got)
	if len(got) != expectedReplay {
		t.Errorf("replayed %d transitions, want %d (this isolates the Go EvtSubscribe binding, not live delivery)", len(got), expectedReplay)
	}
}

// TestBookmarkRoundTripsThroughXML isolates whether EvtCreateBookmark(renderedXML)
// actually reconstructs a usable bookmark, independent of EvtSubscribe entirely.
func TestBookmarkRoundTripsThroughXML(t *testing.T) {
	hist, err := QueryHistory()
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if hist.Bookmark == "" {
		t.Skip("no bookmark produced (no history)")
	}
	t.Logf("bookmark XML: %s", hist.Bookmark)

	bm, err := evtCreateBookmark(hist.Bookmark)
	if err != nil {
		t.Fatalf("evtCreateBookmark(renderedXML) failed: %v", err)
	}
	defer evtClose(bm)

	// Re-render it and see if it round-trips to something equivalent.
	again, err := evtRenderBookmarkXML(bm)
	if err != nil {
		t.Fatalf("re-render bookmark: %v", err)
	}
	t.Logf("re-rendered:  %s", again)
}

// TestWaitForSignalFiresAfterSubscribeWithExistingBacklog checks the raw primitive:
// does WaitForSingleObject ever return signaled at all after EvtSubscribe is given a
// bookmark with events waiting after it? This isolates the wait/reset mechanism from
// EvtNext and from Transition parsing.
func TestWaitForSignalFiresAfterSubscribeWithExistingBacklog(t *testing.T) {
	hist, err := QueryHistory()
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if hist.Bookmark == "" || len(hist.Transitions) < 2 {
		t.Skip("not enough history")
	}

	// Seed a bookmark a few events back from the newest so there is definitely
	// backlog after it, using the SAME mid-history anchoring as the replay test.
	q, err := evtQuery(Channel, XPath, false)
	if err != nil {
		t.Fatalf("evtQuery: %v", err)
	}
	defer evtClose(q)
	handles := make([]evtHandle, batchSize)
	var all []evtHandle
	for {
		batch, err := evtNext(q, handles, 0)
		if err != nil {
			t.Fatalf("evtNext: %v", err)
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
	}
	defer func() {
		for _, h := range all {
			evtClose(h)
		}
	}()
	if len(all) < 2 {
		t.Skip("not enough handles")
	}
	anchorIdx := len(all) - 2
	bm, err := evtCreateBookmark("")
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}
	defer evtClose(bm)
	if err := evtUpdateBookmark(bm, all[anchorIdx]); err != nil {
		t.Fatalf("update bookmark: %v", err)
	}

	sig, err := createManualResetEvent()
	if err != nil {
		t.Fatalf("createManualResetEvent: %v", err)
	}
	defer closeHandle(sig)

	subHandle, err := evtSubscribe(Channel, XPath, bm, sig, evtSubscribeStartAfterBookmark)
	if err != nil {
		t.Fatalf("evtSubscribe: %v", err)
	}
	defer evtClose(subHandle)

	signaled := waitForSignal(sig, 5000)
	t.Logf("waitForSignal (5s) = %v", signaled)
	if !signaled {
		t.Fatal("SignalEvent never fired even though 1 event was already queued past the bookmark")
	}

	out := make([]evtHandle, 8)
	batch, err := evtNext(subHandle, out, 0)
	if err != nil {
		t.Fatalf("EvtNext after signal: %v", err)
	}
	t.Logf("EvtNext returned %d handle(s)", len(batch))
	for _, h := range batch {
		xmlStr, _ := evtRenderEventXML(h)
		t.Logf("  event xml: %s", xmlStr)
		evtClose(h)
	}
}
