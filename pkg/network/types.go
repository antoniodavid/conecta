package network

import "time"

// PortalStatus represents the state of the captive portal
type PortalStatus int

const (
	PortalNone PortalStatus = iota
	PortalNeedsAuth
	PortalConnected
	PortalError
)

func (s PortalStatus) String() string {
	switch s {
	case PortalNone:
		return "no portal"
	case PortalNeedsAuth:
		return "needs auth"
	case PortalConnected:
		return "connected"
	case PortalError:
		return "error"
	default:
		return "unknown"
	}
}

// Connection represents the current network connection state
type Connection struct {
	Status    PortalStatus
	Username  string
	Gateway   string
	Interface string
	IP        string
	PortalURL string
	LastError error
	LastCheck time.Time
}

// SpeedResult holds speed test results
type SpeedResult struct {
	DownloadMbps float64
	UploadMbps   float64
	LatencyMs    float64
	BytesDownload uint64
	Duration     time.Duration
	ServerURL    string
	Error        error
}

// NetworkConfig holds network configuration
type NetworkConfig struct {
	Gateway      string        `yaml:"gateway"`
	Interface    string        `yaml:"interface"`
	PortalURL    string        `yaml:"portal_url"`
	Timeout      time.Duration `yaml:"timeout"`
	RetryCount   int           `yaml:"retry_count"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
}

// DefaultConfig returns default network configuration
func DefaultConfig() *NetworkConfig {
	return &NetworkConfig{
		Gateway:    "192.168.1.1",
		Interface:  "enp3s0",
		PortalURL:  "https://secure.etecsa.net:8443",
		Timeout:    10 * time.Second,
		RetryCount: 999,
		RetryDelay: 15 * time.Second,
	}
}
