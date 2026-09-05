package vpn

// Profile describes one NetworkManager wireguard connection profile.
type Profile struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Device  string `json:"device"`
	IP      string `json:"ip"`
	Country string `json:"country"`
	Flag    string `json:"flag"`
}

// Status represents the current VPN status
type Status struct {
	Connected      bool
	IP             string
	ConnectionName string
	Interface      string
	// Profiles lists all known wireguard profiles (best-effort, may be empty).
	Profiles []Profile
}

// Config holds VPN configuration
type Config struct {
	Enabled   bool   `yaml:"enabled"`
	Interface string `yaml:"interface"`
	Name      string `yaml:"name"`
}

// DefaultConfig returns default VPN configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:   false,
		Interface: "wg0",
		Name:      "USA",
	}
}
