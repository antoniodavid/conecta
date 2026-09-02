package vpn

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		vpnName   string
		wantIface string
		wantName  string
	}{
		{
			name:      "uses provided values",
			iface:     "wg1",
			vpnName:   "Spain",
			wantIface: "wg1",
			wantName:  "Spain",
		},
		{
			name:      "defaults to wg0 and USA",
			iface:     "",
			vpnName:   "",
			wantIface: "wg0",
			wantName:  "USA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.iface, tt.vpnName)
			if m.iface != tt.wantIface {
				t.Errorf("NewManager().iface = %q, want %q", m.iface, tt.wantIface)
			}
			if m.name != tt.wantName {
				t.Errorf("NewManager().name = %q, want %q", m.name, tt.wantName)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled != false {
		t.Errorf("DefaultConfig().Enabled = %v, want %v", cfg.Enabled, false)
	}
	if cfg.Interface != "wg0" {
		t.Errorf("DefaultConfig().Interface = %q, want %q", cfg.Interface, "wg0")
	}
	if cfg.Name != "USA" {
		t.Errorf("DefaultConfig().Name = %q, want %q", cfg.Name, "USA")
	}
}
