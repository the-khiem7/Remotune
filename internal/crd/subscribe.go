//go:build windows

package crd

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type Subscription struct {
	handle      evtHandle
	signalEvent windows.Handle
}

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
func (s *Subscription) Close() {
	if s == nil {
		return
	}
	evtClose(s.handle)
	s.handle = 0
	closeHandle(s.signalEvent)
	s.signalEvent = 0
}
func (s *Subscription) Poll(max int, timeoutMs uint32) ([]Transition, int, error) {
	if s == nil || s.handle == 0 {
		return nil, 0, fmt.Errorf("poll on closed subscription")
	}
	if max <= 0 {
		max = batchSize
	}
	if !waitForSignal(s.signalEvent, timeoutMs) {
		return nil, 0, nil
	}

	handles := make([]evtHandle, max)
	batch, err := evtNext(s.handle, handles, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("poll subscription: %w", err)
	}
	if len(batch) < max {
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
