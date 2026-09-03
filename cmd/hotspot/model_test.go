package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RED 2.4: CLIENTS header must render the formatted count, not a literal %d.
func TestViewFormatsClientCount(t *testing.T) {
	m := NewModel()
	view := m.View()
	if strings.Contains(view, "CLIENTS (%d)") {
		t.Fatalf("view must not contain a literal %%d:\n%s", view)
	}
	if !strings.Contains(view, "CLIENTS (0)") {
		t.Fatalf("view must render the client count:\n%s", view)
	}
}

// RED 2.4: standalone hotspot TUI must load user config, not package defaults.
func TestNewModelLoadsUserConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ".config", "conecta")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "hotspot:\n  subnet: 10.9.9.0/24\nvpn:\n  interface: wg9\nnetwork:\n  interface: eth9\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	m := NewModel()
	if m.cfg == nil {
		t.Fatalf("NewModel must retain the loaded config")
	}
	if m.cfg.Hotspot.Subnet != "10.9.9.0/24" {
		t.Fatalf("hotspot subnet = %q, want user-configured 10.9.9.0/24", m.cfg.Hotspot.Subnet)
	}
	if m.cfg.VPN.Interface != "wg9" {
		t.Fatalf("vpn interface = %q, want user-configured wg9", m.cfg.VPN.Interface)
	}
	if m.cfg.Network.Interface != "eth9" {
		t.Fatalf("network interface = %q, want user-configured eth9", m.cfg.Network.Interface)
	}
}
