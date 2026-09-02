package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Network.Gateway != "192.168.1.1" {
		t.Errorf("DefaultConfig().Network.Gateway = %q, want %q", cfg.Network.Gateway, "192.168.1.1")
	}
	if cfg.Network.Interface != "enp3s0" {
		t.Errorf("DefaultConfig().Network.Interface = %q, want %q", cfg.Network.Interface, "enp3s0")
	}
	if cfg.Hotspot.SSID != "RUBAN_WIFI" {
		t.Errorf("DefaultConfig().Hotspot.SSID = %q, want %q", cfg.Hotspot.SSID, "RUBAN_WIFI")
	}
	if cfg.VPN.Interface != "wg0" {
		t.Errorf("DefaultConfig().VPN.Interface = %q, want %q", cfg.VPN.Interface, "wg0")
	}
	if cfg.UI.RefreshSec != 2 {
		t.Errorf("DefaultConfig().UI.RefreshSec = %d, want %d", cfg.UI.RefreshSec, 2)
	}
}

func TestLoad_ExistingFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := `
network:
  gateway: 10.0.0.1
  interface: eth0
  portal_url: https://example.com

hotspot:
  ssid: TEST_WIFI
  passphrase: test123
  channel: "6"
  freq_band: "5"
  method: bridge
  subnet: 10.0.0.0/24
  gateway: 10.0.0.1

vpn:
  enabled: true
  interface: wg1
  name: Spain

ui:
  theme: dark
  refresh_sec: 5
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Network.Gateway != "10.0.0.1" {
		t.Errorf("Load().Network.Gateway = %q, want %q", cfg.Network.Gateway, "10.0.0.1")
	}
	if cfg.Network.Interface != "eth0" {
		t.Errorf("Load().Network.Interface = %q, want %q", cfg.Network.Interface, "eth0")
	}
	if cfg.Hotspot.SSID != "TEST_WIFI" {
		t.Errorf("Load().Hotspot.SSID = %q, want %q", cfg.Hotspot.SSID, "TEST_WIFI")
	}
	if cfg.Hotspot.Method != "bridge" {
		t.Errorf("Load().Hotspot.Method = %q, want %q", cfg.Hotspot.Method, "bridge")
	}
	if cfg.VPN.Enabled != true {
		t.Errorf("Load().VPN.Enabled = %v, want %v", cfg.VPN.Enabled, true)
	}
	if cfg.VPN.Name != "Spain" {
		t.Errorf("Load().VPN.Name = %q, want %q", cfg.VPN.Name, "Spain")
	}
	if cfg.UI.RefreshSec != 5 {
		t.Errorf("Load().UI.RefreshSec = %d, want %d", cfg.UI.RefreshSec, 5)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should return defaults
	if cfg.Network.Gateway != "192.168.1.1" {
		t.Errorf("Load() with nonexistent file should return defaults, got Gateway = %q", cfg.Network.Gateway)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Network.Gateway = "172.16.0.1"
	cfg.Hotspot.SSID = "MY_WIFI"

	if err := Save(cfg, cfgPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and verify
	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Network.Gateway != "172.16.0.1" {
		t.Errorf("After Save/Load, Gateway = %q, want %q", loaded.Network.Gateway, "172.16.0.1")
	}
	if loaded.Hotspot.SSID != "MY_WIFI" {
		t.Errorf("After Save/Load, SSID = %q, want %q", loaded.Hotspot.SSID, "MY_WIFI")
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()

	if path == "" {
		t.Error("GetConfigPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("GetConfigPath() returned relative path: %q", path)
	}
}
