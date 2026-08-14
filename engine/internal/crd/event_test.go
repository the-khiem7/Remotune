//go:build windows

package crd

import (
	"testing"
)

// Real XML samples captured during the Phase 0 spike (2026-08-14), redacted here to a
// placeholder account. Verifies the parser against actual chromoting Event Log output,
// not an assumed shape.
const (
	sampleConnectXML = `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='chromoting'/><EventID Qualifiers='16384'>1</EventID><Version>0</Version><Level>4</Level><Task>1</Task><Opcode>0</Opcode><Keywords>0x80000000000000</Keywords><TimeCreated SystemTime='2026-08-13T10:53:14.9405943Z'/><EventRecordID>47496</EventRecordID><Correlation/><Execution ProcessID='5996' ThreadID='0'/><Channel>Application</Channel><Computer>THEKHIEM7-TUF</Computer><Security/></System><EventData><Data>khiemnguyen120216@gmail.com/chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a</Data></EventData></Event>`

	sampleDisconnectXML = `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='chromoting'/><EventID Qualifiers='16384'>2</EventID><Version>0</Version><Level>4</Level><Task>1</Task><Opcode>0</Opcode><Keywords>0x80000000000000</Keywords><TimeCreated SystemTime='2026-08-13T11:01:05.5879194Z'/><EventRecordID>47503</EventRecordID><Correlation/><Execution ProcessID='5996' ThreadID='0'/><Channel>Application</Channel><Computer>THEKHIEM7-TUF</Computer><Security/></System><EventData><Data>khiemnguyen120216@gmail.com/chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a</Data></EventData></Event>`

	sampleChannelInfoXML = `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='chromoting'/><EventID Qualifiers='16384'>4</EventID><Version>0</Version><Level>4</Level><Task>1</Task><Opcode>0</Opcode><Keywords>0x80000000000000</Keywords><TimeCreated SystemTime='2026-08-13T10:53:16.1877621Z'/><EventRecordID>47497</EventRecordID><Correlation/><Execution ProcessID='5996' ThreadID='0'/><Channel>Application</Channel><Computer>THEKHIEM7-TUF</Computer><Security/></System><EventData><Data>khiemnguyen120216@gmail.com/chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a</Data><Data>unknown</Data><Data>10.5.0.13:53293</Data><Data></Data><Data>relay</Data></EventData></Event>`
)

func TestParseTransitionConnect(t *testing.T) {
	tr, err := ParseTransition(sampleConnectXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Kind != KindConnected {
		t.Errorf("kind = %v, want Connected", tr.Kind)
	}
	if tr.RecordID != 47496 {
		t.Errorf("record id = %d, want 47496", tr.RecordID)
	}
	if tr.ProcessID != 5996 {
		t.Errorf("process id = %d, want 5996", tr.ProcessID)
	}
	if tr.SessionID != "chromoting_ftl_1486e878-6587-479c-9e07-d3691b14273a" {
		t.Errorf("session id = %q, unexpected", tr.SessionID)
	}
	if tr.Time.Year() != 2026 || tr.Time.Month() != 8 || tr.Time.Day() != 13 {
		t.Errorf("time = %v, unexpected", tr.Time)
	}
}

func TestParseTransitionDisconnect(t *testing.T) {
	tr, err := ParseTransition(sampleDisconnectXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Kind != KindDisconnected {
		t.Errorf("kind = %v, want Disconnected", tr.Kind)
	}
	if tr.RecordID != 47503 {
		t.Errorf("record id = %d, want 47503", tr.RecordID)
	}
}

func TestParseTransitionConnectDisconnectShareSessionID(t *testing.T) {
	c, err := ParseTransition(sampleConnectXML)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	d, err := ParseTransition(sampleDisconnectXML)
	if err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	if c.SessionID != d.SessionID {
		t.Errorf("connect session %q != disconnect session %q; ledger decision 34 requires they match", c.SessionID, d.SessionID)
	}
}

// TestParseTransitionChannelInfoIsExcluded verifies event 4 (channel info, which
// carries the client IP) is rejected by ParseTransition rather than silently parsed.
// The detector's XPath already excludes it; this guards against a caller feeding it
// event 4 XML by mistake and accidentally retaining an IP address.
func TestParseTransitionChannelInfoIsExcluded(t *testing.T) {
	_, err := ParseTransition(sampleChannelInfoXML)
	if err == nil {
		t.Fatal("expected an error for a non-transition (channel info) event")
	}
}

func TestParseTransitionRejectsMalformedXML(t *testing.T) {
	if _, err := ParseTransition("not xml at all"); err == nil {
		t.Fatal("expected an error for malformed XML")
	}
}

func TestParseTransitionRejectsWrongProvider(t *testing.T) {
	xml := `<Event><System><Provider Name='SomeOtherProvider'/><EventID>1</EventID><EventRecordID>1</EventRecordID><TimeCreated SystemTime='2026-01-01T00:00:00Z'/></System><EventData><Data>a@b.com/chromoting_ftl_x</Data></EventData></Event>`
	if _, err := ParseTransition(xml); err == nil {
		t.Fatal("expected an error for a non-chromoting provider")
	}
}

// TestXPathMatchesOnlyTransitionEvents documents (and pins) that the detector's own
// query filter never asks for event 4.
func TestXPathMatchesOnlyTransitionEvents(t *testing.T) {
	if XPath != `*[System[Provider[@Name='chromoting'] and (EventID=1 or EventID=2)]]` {
		t.Fatalf("XPath changed unexpectedly: %s", XPath)
	}
}
