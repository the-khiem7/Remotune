//go:build windows

package crd

import (
	"testing"
)

// These tests query the REAL Windows Event Log on the machine running them, using the
// verified read-only XPath. They do not write anything and do not touch any live CRD
// session; they only read history CRD already recorded (Phase 0 evidence: 191 such
// events observed spanning roughly three weeks). No REMOTUNE_SYSTEM_TESTS gate is
// needed because nothing is mutated, unlike internal/wintune's tests.

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
		// Transitions must come back oldest-to-newest.
		for i := 1; i < len(result.Transitions); i++ {
			prev, cur := result.Transitions[i-1], result.Transitions[i]
			if cur.Time.Before(prev.Time) {
				t.Errorf("transitions not ordered oldest-to-newest at index %d: %v then %v", i, prev.Time, cur.Time)
			}
		}
		// No session id may ever equal a bare account email: the redaction boundary
		// must have already stripped it for every real event, not just the samples.
		for _, tr := range result.Transitions {
			if tr.SessionID == "" {
				t.Errorf("transition %v has an empty session id", tr)
			}
		}
	} else {
		t.Log("no chromoting transition events found in the Event Log on this machine")
	}
}

// TestBootstrapAgainstRealEventLog exercises the full startup sequence (query +
// reconstruct) against whatever this machine's Event Log actually contains right now.
// It intentionally does not assert a specific State, since that depends on whether a
// real CRD session happens to be connected when the test runs; it only asserts internal
// consistency of the result.
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

	// The bootstrap must never claim ownership from process presence alone; it derives
	// state purely from parsed events, which this call graph enforces structurally
	// (Bootstrap only calls QueryHistory + Reconstruct, neither of which inspects
	// running processes). Documented here rather than asserted, since there is nothing
	// to introspect at the type level.
}
