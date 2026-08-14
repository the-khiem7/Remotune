//go:build windows

package crd

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// EventKind is the meaning of a chromoting Event Log record, restricted to the two
// transition events this detector acts on. Event 4 (channel info) is parsed only far
// enough to be classified and discarded; it is diagnostic-only per Phase 0 evidence.
type EventKind int

const (
	// KindUnknown is any chromoting event ID this detector does not act on.
	KindUnknown EventKind = iota
	KindConnected
	KindDisconnected
)

func (k EventKind) String() string {
	switch k {
	case KindConnected:
		return "Connected"
	case KindDisconnected:
		return "Disconnected"
	default:
		return "Unknown"
	}
}

// Event IDs verified in Phase 0 for the "chromoting" legacy event source on the
// Application channel.
const (
	eventIDConnected    = 1
	eventIDDisconnected = 2
	eventIDChannelInfo  = 4
)

// Channel and Provider are the verified Event Log coordinates for the CRD detector.
const (
	Channel  = "Application"
	Provider = "chromoting"
)

// XPath is the verified filter selecting only the two transition events. Event 4 is
// deliberately excluded: it is diagnostic-only and carries the client IP, which this
// detector must not query for by default.
const XPath = `*[System[Provider[@Name='` + Provider + `'] and (EventID=1 or EventID=2)]]`

// Transition is one parsed, redacted connect or disconnect record.
//
// SessionID is the JID resource component (chromoting_ftl_<uuid>), unique per session
// and identical across that session's connect and disconnect (ledger decision 34). The
// account email prefix and, for event 4, the client IP are never retained (decision 45).
type Transition struct {
	Kind      EventKind
	RecordID  uint64
	Time      time.Time
	ProcessID uint32
	SessionID string
}

func (t Transition) String() string {
	return fmt.Sprintf("%s rec=%d time=%s pid=%d session=%s", t.Kind, t.RecordID, t.Time.Format(time.RFC3339), t.ProcessID, t.SessionID)
}

// rawEvent mirrors the subset of the Windows Event XML schema this detector needs.
type rawEvent struct {
	XMLName xml.Name `xml:"Event"`
	System  struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID     int `xml:"EventID"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
		EventRecordID uint64 `xml:"EventRecordID"`
		Execution     struct {
			ProcessID uint32 `xml:"ProcessID,attr"`
		} `xml:"Execution"`
		Channel string `xml:"Channel"`
	} `xml:"System"`
	EventData struct {
		Data []string `xml:"Data"`
	} `xml:"EventData"`
}

// ErrNotTransitionEvent is returned by ParseTransition for a well-formed chromoting
// event that is not a connect or disconnect (for example, event 4).
var ErrNotTransitionEvent = fmt.Errorf("event is not a connect/disconnect transition")

// ParseTransition parses one event's rendered XML into a redacted Transition.
//
// Verified against Phase 0 samples such as:
//
//	<Event ...><System><Provider Name='chromoting'/><EventID Qualifiers='16384'>1</EventID>
//	...<EventRecordID>47496</EventRecordID><Execution ProcessID='5996' ThreadID='0'/>
//	...</System><EventData><Data>user@example.com/chromoting_ftl_<uuid></Data></EventData></Event>
func ParseTransition(eventXML string) (Transition, error) {
	var raw rawEvent
	if err := xml.Unmarshal([]byte(eventXML), &raw); err != nil {
		return Transition{}, fmt.Errorf("parse event XML: %w", err)
	}

	if raw.System.Provider.Name != Provider {
		return Transition{}, fmt.Errorf("unexpected provider %q", raw.System.Provider.Name)
	}

	var kind EventKind
	switch raw.System.EventID {
	case eventIDConnected:
		kind = KindConnected
	case eventIDDisconnected:
		kind = KindDisconnected
	default:
		return Transition{}, ErrNotTransitionEvent
	}

	if len(raw.EventData.Data) == 0 {
		return Transition{}, fmt.Errorf("event %d has no EventData", raw.System.EventRecordID)
	}

	sessionID, err := sessionIDFromJID(raw.EventData.Data[0])
	if err != nil {
		return Transition{}, fmt.Errorf("event %d: %w", raw.System.EventRecordID, err)
	}

	ts, err := time.Parse(time.RFC3339Nano, raw.System.TimeCreated.SystemTime)
	if err != nil {
		return Transition{}, fmt.Errorf("event %d: parse TimeCreated: %w", raw.System.EventRecordID, err)
	}

	return Transition{
		Kind:      kind,
		RecordID:  raw.System.EventRecordID,
		Time:      ts,
		ProcessID: raw.System.Execution.ProcessID,
		SessionID: sessionID,
	}, nil
}

// sessionIDFromJID extracts the resource component from a CRD client JID of the form
// "<account>/chromoting_ftl_<uuid>" and discards the account part immediately.
//
// This is the redaction boundary: the account email must never travel past this
// function into any struct that gets logged, persisted, or displayed (ledger decision 45).
func sessionIDFromJID(jid string) (string, error) {
	i := strings.LastIndex(jid, "/")
	if i < 0 || i == len(jid)-1 {
		return "", fmt.Errorf("malformed CRD JID (no resource component)")
	}
	resource := jid[i+1:]
	if !strings.HasPrefix(resource, "chromoting_ftl_") {
		return "", fmt.Errorf("malformed CRD JID (unexpected resource prefix %q)", resource)
	}
	return resource, nil
}
