package crd

import (
	"testing"
	"time"
)

func mustJID(t *testing.T, jid string) string {
	t.Helper()
	sid, err := sessionIDFromJID(jid)
	if err != nil {
		t.Fatalf("sessionIDFromJID(%q): %v", jid, err)
	}
	return sid
}

func tr(kind EventKind, rec uint64, minute int, pid uint32, session string) Transition {
	return Transition{
		Kind:      kind,
		RecordID:  rec,
		Time:      time.Date(2026, 8, 13, 9, minute, 0, 0, time.UTC),
		ProcessID: pid,
		SessionID: session,
	}
}

func TestReconstructEmptyIsUnknown(t *testing.T) {
	s := Reconstruct(nil)
	if s.State != StateUnknown {
		t.Fatalf("state = %v, want Unknown for no history at all", s.State)
	}
}

func TestReconstructNormalConnectDisconnect(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 100, "sessA"),
		tr(KindDisconnected, 2, 5, 100, "sessA"),
	}
	s := Reconstruct(events)
	if s.State != StateDisconnected {
		t.Fatalf("state = %v, want Disconnected after a matched pair", s.State)
	}
	if len(s.ActiveSessions) != 0 {
		t.Fatalf("active sessions = %v, want none", s.ActiveSessions)
	}
	if s.DroppedDisconnects != 0 {
		t.Fatalf("dropped disconnects = %d, want 0", s.DroppedDisconnects)
	}
}

func TestReconstructStillConnected(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 100, "sessA"),
	}
	s := Reconstruct(events)
	if s.State != StateConnected {
		t.Fatalf("state = %v, want Connected", s.State)
	}
	if len(s.ActiveSessions) != 1 || s.ActiveSessions[0].SessionID != "sessA" {
		t.Fatalf("active sessions = %v, want exactly sessA", s.ActiveSessions)
	}
}
func TestReconstructHostRestartDropsStaleConnect(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 22864, "sessA"), // old host lifetime, never disconnected
		tr(KindConnected, 2, 10, 6544, "sessB"), // host restarted; new lifetime begins
	}
	s := Reconstruct(events)
	if s.State != StateConnected {
		t.Fatalf("state = %v, want Connected (sessB is genuinely active)", s.State)
	}
	if len(s.ActiveSessions) != 1 || s.ActiveSessions[0].SessionID != "sessB" {
		t.Fatalf("active sessions = %v, want only sessB; sessA must not survive the host restart", s.ActiveSessions)
	}
	if s.CurrentHostPID != 6544 {
		t.Fatalf("current host PID = %d, want 6544", s.CurrentHostPID)
	}
}
func TestReconstructDisconnectAfterHostRestartIsDropped(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 100, "sessA"),
		tr(KindConnected, 2, 5, 200, "sessB"),    // host restart; sessA's lifetime is gone
		tr(KindDisconnected, 3, 6, 200, "sessA"), // a disconnect for a session that no
	}
	s := Reconstruct(events)
	if s.DroppedDisconnects != 1 {
		t.Fatalf("dropped disconnects = %d, want 1", s.DroppedDisconnects)
	}
	if len(s.ActiveSessions) != 1 || s.ActiveSessions[0].SessionID != "sessB" {
		t.Fatalf("active sessions = %v, want only sessB unaffected", s.ActiveSessions)
	}
}

func TestReconstructDuplicateConnectIsIdempotent(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 100, "sessA"),
		tr(KindConnected, 2, 1, 100, "sessA"), // duplicate connect, same session/PID
	}
	s := Reconstruct(events)
	if s.State != StateConnected || len(s.ActiveSessions) != 1 {
		t.Fatalf("duplicate connect must not create two entries: %v", s.ActiveSessions)
	}
}

func TestReconstructDuplicateDisconnectDoesNotCorrupt(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 100, "sessA"),
		tr(KindDisconnected, 2, 1, 100, "sessA"),
		tr(KindDisconnected, 3, 2, 100, "sessA"), // duplicate disconnect
	}
	s := Reconstruct(events)
	if s.State != StateDisconnected {
		t.Fatalf("state = %v, want Disconnected", s.State)
	}
	if s.DroppedDisconnects != 1 {
		t.Fatalf("dropped disconnects = %d, want 1 for the duplicate", s.DroppedDisconnects)
	}
}
func TestReconstructMultipleConcurrentSessions(t *testing.T) {
	events := []Transition{
		tr(KindConnected, 1, 0, 100, "sessA"),
		tr(KindConnected, 2, 1, 100, "sessB"),
		tr(KindDisconnected, 3, 2, 100, "sessA"),
	}
	s := Reconstruct(events)
	if s.State != StateConnected {
		t.Fatalf("state = %v, want Connected (sessB still active)", s.State)
	}
	if len(s.ActiveSessions) != 1 || s.ActiveSessions[0].SessionID != "sessB" {
		t.Fatalf("active sessions = %v, want only sessB", s.ActiveSessions)
	}
}

func TestReconstructOutOfOrderInput(t *testing.T) {
	events := []Transition{
		tr(KindDisconnected, 2, 5, 100, "sessA"),
		tr(KindConnected, 1, 0, 100, "sessA"),
	}
	s := Reconstruct(events)
	if s.State != StateDisconnected {
		t.Fatalf("state = %v, want Disconnected once correctly ordered", s.State)
	}
}

func TestSessionIDFromJIDRedactsAccount(t *testing.T) {
	sid, err := sessionIDFromJID("someone@example.com/chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sid != "chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a" {
		t.Fatalf("session id = %q, want the resource component only", sid)
	}
	if sid == "someone@example.com/chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a" {
		t.Fatal("account email leaked into the session id")
	}
}

func TestSessionIDFromJIDRejectsMalformed(t *testing.T) {
	cases := []string{"", "no-slash-here", "user@example.com/", "user@example.com/not_chromoting"}
	for _, c := range cases {
		if _, err := sessionIDFromJID(c); err == nil {
			t.Errorf("sessionIDFromJID(%q) should have failed", c)
		}
	}
}
