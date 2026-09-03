package hotspot

// Config holds hotspot configuration
type Config struct {
	SSID       string
	Passphrase string
	Channel    string
	FreqBand   string
	Method     string
	Subnet     string
	Gateway    string
	InternetInterface string
}

// DefaultConfig returns default hotspot configuration
func DefaultConfig() *Config {
	return &Config{
		SSID:       "RUBAN_WIFI",
		Passphrase: "CHANGE-ME-USE-16-PLUS-CHARS",
		Channel:    "default",
		FreqBand:   "2.4",
		Method:     "nat",
		Subnet:     "192.168.12.0/24",
		Gateway:    "192.168.12.1",
	}
}

// Status represents the current hotspot status
type Status struct {
	Active   bool
	SSID     string
	IP       string
	Channel  int
	FreqBand string
	Method   string
	Hostapd  bool
	Dnsmasq  bool
	Clients  int
}
