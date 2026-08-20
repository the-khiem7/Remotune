package crd

// This file's reconstruction logic is pure and platform-independent (no build tag),
// so it can be unit tested without touching the Windows Event Log.

import "sort"

// State is the detector's observed connection state.
type State int

const (
	StateUnknown State = iota
	StateDisconnected
	StateConnected
)

func (s State) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnected:
		return "Connected"
	default:
		return "Unknown"
	}
}

// Session is one entry in the active-client set: a connect that has not been matched
// by a disconnect within the current host process lifetime.
type Session struct {
	SessionID   string
	ProcessID   uint32
	ConnectedAt Transition
}

// Snapshot is the reconstructed detector state at a point in time.
type Snapshot struct {
	State          State
	ActiveSessions []Session
	// LastTransition is the most recent redacted transition used to derive this
	// snapshot. It lets operator diagnostics distinguish an honestly observed
	// disconnect from an unknown/stale detector state without retaining EventData.
	LastTransition    Transition
	HasLastTransition bool
	// CurrentHostPID is the ProcessID of the most recent transition observed, used to
	// scope reconstruction to the current host process lifetime (ledger decision 36).
	// Zero if no transitions were observed at all.
	CurrentHostPID uint32
	// DroppedDisconnects counts disconnect events that referenced a session not in the
	// active set, which happens when the matching connect belonged to an earlier host
	// process lifetime and was already discarded. Diagnostic only.
	DroppedDisconnects int
}

// Reconstruct derives the current detector state from an ordered (oldest-to-newest)
// sequence of transitions.
//
// Core rule, established in Phase 0 (ledger decisions 35 and 36): connect and
// disconnect do NOT reliably alternate. A disconnect is genuinely lost when the CRD
// host process dies. Reconstruction is therefore scoped to the current host process
// lifetime: whenever a transition's ProcessID differs from the running host PID, the
// active set from the previous lifetime is discarded rather than carried forward,
// because a connect left dangling by a dead host process must be treated as
// disconnected, never as an active session.
//
// Sessions are keyed by SessionID (the per-session JID resource, unique across a
// session's connect and disconnect per decision 34), which is what allows more than
// one concurrent session to be tracked correctly if CRD ever exhibits that behavior,
// without assuming it does (decision remains UNVERIFIED at the product level).
func Reconstruct(transitions []Transition) Snapshot {
	sorted := make([]Transition, len(transitions))
	copy(sorted, transitions)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Time.Equal(sorted[j].Time) {
			return sorted[i].RecordID < sorted[j].RecordID
		}
		return sorted[i].Time.Before(sorted[j].Time)
	})

	active := map[string]Session{}
	var currentPID uint32
	var dropped int

	for _, t := range sorted {
		if currentPID != 0 && t.ProcessID != currentPID {
			// Host process lifetime changed. Anything still "active" belonged to a
			// process that is gone; it cannot still be connected.
			active = map[string]Session{}
		}
		currentPID = t.ProcessID

		switch t.Kind {
		case KindConnected:
			active[t.SessionID] = Session{
				SessionID:   t.SessionID,
				ProcessID:   t.ProcessID,
				ConnectedAt: t,
			}
		case KindDisconnected:
			if _, ok := active[t.SessionID]; ok {
				delete(active, t.SessionID)
			} else {
				dropped++
			}
		}
	}

	state := StateDisconnected
	if len(sorted) == 0 {
		state = StateUnknown
	} else if len(active) > 0 {
		state = StateConnected
	}

	sessions := make([]Session, 0, len(active))
	for _, s := range active {
		sessions = append(sessions, s)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].SessionID < sessions[j].SessionID })

	snapshot := Snapshot{
		State:              state,
		ActiveSessions:     sessions,
		CurrentHostPID:     currentPID,
		DroppedDisconnects: dropped,
	}
	if len(sorted) > 0 {
		snapshot.LastTransition = sorted[len(sorted)-1]
		snapshot.HasLastTransition = true
	}
	return snapshot
}
