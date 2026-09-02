package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config holds all application configuration
type Config struct {
	Network     NetworkConfig  `yaml:"network"`
	Hotspot     HotspotConfig  `yaml:"hotspot"`
	VPN         VPNConfig      `yaml:"vpn"`
	UI          UIConfig       `yaml:"ui"`
	Credentials Credentials    `yaml:"credentials"`
}

// Credentials holds login credentials
type Credentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// NetworkConfig holds network settings
type NetworkConfig struct {
	Gateway    string `yaml:"gateway"`
	Interface  string `yaml:"interface"`
	PortalURL  string `yaml:"portal_url"`
}

// HotspotConfig holds hotspot settings
type HotspotConfig struct {
	SSID       string `yaml:"ssid"`
	Passphrase string `yaml:"passphrase"`
	Channel    string `yaml:"channel"`
	FreqBand   string `yaml:"freq_band"`
	Method     string `yaml:"method"`
	Subnet     string `yaml:"subnet"`
	Gateway    string `yaml:"gateway"`
}

// VPNConfig holds VPN settings
type VPNConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Interface string `yaml:"interface"`
	Name      string `yaml:"name"`
}

// UIConfig holds UI settings
type UIConfig struct {
	Theme      string `yaml:"theme"`
	RefreshSec int    `yaml:"refresh_sec"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			Gateway:   "192.168.1.1",
			Interface: "enp3s0",
			PortalURL: "https://secure.etecsa.net:8443",
		},
		Hotspot: HotspotConfig{
			SSID:       "RUBAN_WIFI",
			Passphrase: "Bunker.871217",
			Channel:    "default",
			FreqBand:   "2.4",
			Method:     "nat",
			Subnet:     "192.168.12.0/24",
			Gateway:    "192.168.12.1",
		},
		VPN: VPNConfig{
			Enabled:   false,
			Interface: "wg0",
			Name:      "USA",
		},
		UI: UIConfig{
			Theme:      "default",
			RefreshSec: 2,
		},
	}
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves configuration to file
func Save(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "conecta", "config.yaml")
}
