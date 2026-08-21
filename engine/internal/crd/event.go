//go:build windows

package crd

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type EventKind int

const (
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

const (
	eventIDConnected    = 1
	eventIDDisconnected = 2
	eventIDChannelInfo  = 4
)
const (
	Channel  = "Application"
	Provider = "chromoting"
)
const XPath = `*[System[Provider[@Name='` + Provider + `'] and (EventID=1 or EventID=2)]]`

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

var ErrNotTransitionEvent = fmt.Errorf("event is not a connect/disconnect transition")

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
