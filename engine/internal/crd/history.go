//go:build windows

package crd

import (
	"fmt"
)

// HistoryResult is the outcome of a historical bootstrap query.
type HistoryResult struct {
	// Transitions are ordered oldest to newest.
	Transitions []Transition
	// Bookmark is the rendered bookmark XML positioned at the last (newest) event
	// consumed, or "" if no events were found at all. Seeding EvtSubscribe with this
	// bookmark and read-existing-events enabled is the gap-free handover verified in
	// Phase 0 (ledger decision 37): no transition between the historical read and the
	// live subscription can be lost.
	Bookmark string
	// SkippedMalformed counts events that matched the XPath filter but could not be
	// parsed (for example, a session ID that failed validation). They are dropped
	// rather than treated as a fatal query error, but the count is surfaced so a
	// caller can log or alert on it instead of silently losing data.
	SkippedMalformed int
}

// batchSize bounds how many event handles EvtNext is asked to return per call.
const batchSize = 32

// QueryHistory performs the mandatory startup step: read all matching historical
// events from the channel, oldest to newest, parse each into a Transition, and return
// a bookmark positioned at the newest event consumed.
//
// This is read-only. It replays what the Windows Event Log already recorded and does
// not touch any live CRD session.
func QueryHistory() (HistoryResult, error) {
	q, err := evtQuery(Channel, XPath, false)
	if err != nil {
		return HistoryResult{}, fmt.Errorf("query history: %w", err)
	}
	defer evtClose(q)

	var result HistoryResult
	var newest evtHandle // kept open until the bookmark is positioned on it

	handles := make([]evtHandle, batchSize)
	for {
		batch, err := evtNext(q, handles, 0)
		if err != nil {
			evtClose(newest)
			return result, fmt.Errorf("read history batch: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		for i, h := range batch {
			xmlStr, rerr := evtRenderEventXML(h)
			if rerr != nil {
				evtClose(h)
				evtClose(newest)
				return result, fmt.Errorf("render history event: %w", rerr)
			}
			t, perr := ParseTransition(xmlStr)
			if perr != nil {
				result.SkippedMalformed++
			} else {
				result.Transitions = append(result.Transitions, t)
			}

			// The newest event overall is the last handle of the last batch. Every
			// other handle, including a previous "newest", is closed immediately.
			isLastOfBatch := i == len(batch)-1
			if isLastOfBatch {
				evtClose(newest)
				newest = h
			} else {
				evtClose(h)
			}
		}
	}

	if newest == 0 {
		// No events at all: still return a valid, unpositioned bookmark so the caller
		// can subscribe from "now" without a special case.
		result.Bookmark, err = renderNewBookmark(0)
		return result, err
	}
	defer evtClose(newest)

	result.Bookmark, err = renderNewBookmark(newest)
	if err != nil {
		return result, err
	}
	return result, nil
}

// renderNewBookmark creates a fresh bookmark, optionally positions it on event, and
// returns its rendered XML. event may be 0, producing an unpositioned bookmark.
func renderNewBookmark(event evtHandle) (string, error) {
	bm, err := evtCreateBookmark("")
	if err != nil {
		return "", fmt.Errorf("create bookmark: %w", err)
	}
	defer evtClose(bm)

	if event != 0 {
		if err := evtUpdateBookmark(bm, event); err != nil {
			return "", fmt.Errorf("position bookmark: %w", err)
		}
	}

	xmlStr, err := evtRenderBookmarkXML(bm)
	if err != nil {
		return "", fmt.Errorf("render bookmark: %w", err)
	}
	return xmlStr, nil
}
