//go:build windows

package crd

import (
	"fmt"
)

type HistoryResult struct {
	Transitions      []Transition
	Bookmark         string
	SkippedMalformed int
}

const batchSize = 32

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
