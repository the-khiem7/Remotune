//go:build windows

package crd

import (
	"testing"
	"time"
)

func TestSubscribeReplaysFromMidHistoryBookmark(t *testing.T) {
	hist, err := QueryHistory()
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if len(hist.Transitions) < 6 {
		t.Skip("not enough history on this machine to pick a mid-point bookmark")
	}
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
	again, err := evtRenderBookmarkXML(bm)
	if err != nil {
		t.Fatalf("re-render bookmark: %v", err)
	}
	t.Logf("re-rendered:  %s", again)
}
func TestWaitForSignalFiresAfterSubscribeWithExistingBacklog(t *testing.T) {
	hist, err := QueryHistory()
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if hist.Bookmark == "" || len(hist.Transitions) < 2 {
		t.Skip("not enough history")
	}
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
