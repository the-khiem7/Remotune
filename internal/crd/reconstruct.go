package crd

import "sort"

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

type Session struct {
	SessionID   string
	ProcessID   uint32
	ConnectedAt Transition
}
type Snapshot struct {
	State              State
	ActiveSessions     []Session
	LastTransition     Transition
	HasLastTransition  bool
	CurrentHostPID     uint32
	DroppedDisconnects int
}

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
