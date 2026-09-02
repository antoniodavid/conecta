package network

import (
	"testing"
)

func TestExtractInput(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		field    string
		expected string
	}{
		{
			name:     "extracts value from name then value",
			html:     `<input name="username" value="test123">`,
			field:    "username",
			expected: "test123",
		},
		{
			name:     "extracts value from value then name",
			html:     `<input value="test123" name="username">`,
			field:    "username",
			expected: "test123",
		},
		{
			name:     "extracts hidden field",
			html:     `<input type="hidden" name="CSRFHW" value="abc123">`,
			field:    "CSRFHW",
			expected: "abc123",
		},
		{
			name:     "returns empty for missing field",
			html:     `<input name="other" value="test">`,
			field:    "username",
			expected: "",
		},
		{
			name:     "returns empty for empty html",
			html:     "",
			field:    "username",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractInput(tt.html, tt.field)
			if result != tt.expected {
				t.Errorf("extractInput(%q, %q) = %q, want %q", tt.html, tt.field, result, tt.expected)
			}
		})
	}
}

func TestPortalStatus_String(t *testing.T) {
	tests := []struct {
		status   PortalStatus
		expected string
	}{
		{PortalNone, "no portal"},
		{PortalNeedsAuth, "needs auth"},
		{PortalConnected, "connected"},
		{PortalError, "error"},
		{PortalStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("PortalStatus.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway != "192.168.1.1" {
		t.Errorf("DefaultConfig().Gateway = %q, want %q", cfg.Gateway, "192.168.1.1")
	}
	if cfg.Interface != "enp3s0" {
		t.Errorf("DefaultConfig().Interface = %q, want %q", cfg.Interface, "enp3s0")
	}
	if cfg.PortalURL != "https://secure.etecsa.net:8443" {
		t.Errorf("DefaultConfig().PortalURL = %q, want %q", cfg.PortalURL, "https://secure.etecsa.net:8443")
	}
}
