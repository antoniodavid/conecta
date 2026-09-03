package network

import (
	"testing"
)

// RED 2.4/3.2: link snapshot on a missing interface must report unavailable,
// never zero-signal success.
func TestLinkSnapshotMissingInterfaceUnavailable(t *testing.T) {
	snap := SnapshotLink("conecta-nonexistent-iface-xyz")
	if snap.Available {
		t.Fatalf("missing interface must be unavailable, got %+v", snap)
	}
}

// RED 2.4/3.2: hostile SSIDs survive parsing verbatim for JSON escaping upstream.
func TestParseIWLinkSSIDHostileVerbatim(t *testing.T) {
	out := "Connected to aa:bb:cc:dd:ee:ff\n\tSSID: a\"b\\c$(evil)\n\tsignal: -70 dBm\n"
	if got := ParseIWLinkSSID(out); got != `a"b\c$(evil)` {
		t.Fatalf("hostile SSID must be preserved verbatim, got %q", got)
	}
}

func TestParseIWLinkSSIDEmpty(t *testing.T) {
	if got := ParseIWLinkSSID("Not connected.\n"); got != "" {
		t.Fatalf("disconnected output must yield empty SSID, got %q", got)
	}
}
