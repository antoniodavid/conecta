package vpn

// Status represents the current VPN status
type Status struct {
	Connected      bool
	IP             string
	ConnectionName string
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
