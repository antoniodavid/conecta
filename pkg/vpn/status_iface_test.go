package vpn

import (
	"testing"
)

// RED 2.4/3.2: VPN status must carry the configured interface so adapters
// stop hardcoding wg0.
func TestStatusCarriesConfiguredInterface(t *testing.T) {
	m := NewManager("wg9", "Test")
	s, err := m.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if s.Interface != "wg9" {
		t.Fatalf("Status().Interface = %q, want configured wg9", s.Interface)
	}
}
