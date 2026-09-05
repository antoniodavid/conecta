package vpn

import (
	"testing"
)

// VPN status carries the configured interface when no connection is found,
// and adopts the actual UP wireguard iface (e.g. USA instead of wg0) when the
// fallback scan finds one.
func TestStatusCarriesConfiguredInterface(t *testing.T) {
	m := NewManager("wg9", "Test")
	s, err := m.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if s.Connected {
		// Fallback adopted a live UP wireguard iface carrying IPv4.
		if s.Interface == "" {
			t.Fatalf("connected Status().Interface must not be empty")
		}
		if s.IP == "" {
			t.Fatalf("connected Status().IP must not be empty")
		}
		return
	}
	if s.Interface != "wg9" {
		t.Fatalf("disconnected Status().Interface = %q, want configured wg9", s.Interface)
	}
}
