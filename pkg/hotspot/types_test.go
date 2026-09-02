package hotspot

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.SSID != "RUBAN_WIFI" {
		t.Errorf("DefaultConfig().SSID = %q, want %q", cfg.SSID, "RUBAN_WIFI")
	}
	if cfg.Passphrase != "Bunker.871217" {
		t.Errorf("DefaultConfig().Passphrase = %q, want %q", cfg.Passphrase, "Bunker.871217")
	}
	if cfg.Channel != "default" {
		t.Errorf("DefaultConfig().Channel = %q, want %q", cfg.Channel, "default")
	}
	if cfg.FreqBand != "2.4" {
		t.Errorf("DefaultConfig().FreqBand = %q, want %q", cfg.FreqBand, "2.4")
	}
	if cfg.Method != "nat" {
		t.Errorf("DefaultConfig().Method = %q, want %q", cfg.Method, "nat")
	}
	if cfg.Subnet != "192.168.12.0/24" {
		t.Errorf("DefaultConfig().Subnet = %q, want %q", cfg.Subnet, "192.168.12.0/24")
	}
	if cfg.Gateway != "192.168.12.1" {
		t.Errorf("DefaultConfig().Gateway = %q, want %q", cfg.Gateway, "192.168.12.1")
	}
}

func TestClientManager_NewClientManager(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		wantIface string
	}{
		{
			name:     "uses provided interface",
			iface:    "wlan0",
			wantIface: "wlan0",
		},
		{
			name:     "defaults to ap0",
			iface:    "",
			wantIface: "ap0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewClientManager(tt.iface)
			if cm.iface != tt.wantIface {
				t.Errorf("NewClientManager(%q).iface = %q, want %q", tt.iface, cm.iface, tt.wantIface)
			}
		})
	}
}
