//go:build windows

package crd

import (
	"testing"
)

func TestQueryHistoryAgainstRealEventLog(t *testing.T) {
	result, err := QueryHistory()
	if err != nil {
		t.Fatalf("QueryHistory failed: %v", err)
	}

	t.Logf("transitions=%d skippedMalformed=%d bookmarkLen=%d",
		len(result.Transitions), result.SkippedMalformed, len(result.Bookmark))

	if result.SkippedMalformed > 0 {
		t.Logf("WARNING: %d event(s) matched the XPath filter but failed to parse", result.SkippedMalformed)
	}

	if len(result.Transitions) > 0 {
		if result.Bookmark == "" {
			t.Error("bookmark is empty despite finding transitions; startup handover would be unsafe")
		}
		for i := 1; i < len(result.Transitions); i++ {
			prev, cur := result.Transitions[i-1], result.Transitions[i]
			if cur.Time.Before(prev.Time) {
				t.Errorf("transitions not ordered oldest-to-newest at index %d: %v then %v", i, prev.Time, cur.Time)
			}
		}
		for _, tr := range result.Transitions {
			if tr.SessionID == "" {
				t.Errorf("transition %v has an empty session id", tr)
			}
		}
	} else {
		t.Log("no chromoting transition events found in the Event Log on this machine")
	}
}
func TestBootstrapAgainstRealEventLog(t *testing.T) {
	res, err := Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	t.Logf("state=%s activeSessions=%d currentHostPID=%d droppedDisconnects=%d skippedMalformed=%d",
		res.Snapshot.State, len(res.Snapshot.ActiveSessions), res.Snapshot.CurrentHostPID,
		res.Snapshot.DroppedDisconnects, res.SkippedMalformed)

	switch res.Snapshot.State {
	case StateConnected:
		if len(res.Snapshot.ActiveSessions) == 0 {
			t.Error("state is Connected but no active sessions are recorded")
		}
	case StateDisconnected, StateUnknown:
		if len(res.Snapshot.ActiveSessions) != 0 {
			t.Errorf("state is %s but %d active session(s) are recorded", res.Snapshot.State, len(res.Snapshot.ActiveSessions))
		}
	}
}
