package network

import (
	"errors"
	"strings"
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

func TestClassifyPortalPage(t *testing.T) {
	tests := []struct {
		name string
		html string
		want PortalStatus
	}{
		{name: "session marker ya esta conectado", html: "<div>usted ya está conectado</div>", want: PortalConnected},
		{name: "session marker ya conectado", html: "usuario ya conectado", want: PortalConnected},
		{name: "login marker LoginServlet", html: `<form action="LoginServlet">`, want: PortalNeedsAuth},
		{name: "login marker Bienvenido", html: "<h1>Bienvenido</h1>", want: PortalNeedsAuth},
		{name: "empty page", html: "", want: PortalNone},
		{name: "unknown page", html: "<html><body>gateway notice</body></html>", want: PortalNone},
		{name: "mixed session and login markers", html: `ya está conectado <form action="LoginServlet">`, want: PortalConnected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyPortalPage(tt.html); got != tt.want {
				t.Errorf("classifyPortalPage(%q) = %v, want %v", tt.html, got, tt.want)
			}
		})
	}
}

func TestLogoutVerdict(t *testing.T) {
	tests := []struct {
		name    string
		page    string
		wantErr string // "" means success
	}{
		{name: "login page after logout is success", page: `<form action="LoginServlet">Bienvenido</form>`, wantErr: ""},
		{name: "login marker only is success", page: "LoginServlet", wantErr: ""},
		{name: "session page means still active", page: "usted ya está conectado", wantErr: "portal session still active"},
		{name: "empty page fails closed", page: "", wantErr: "cannot verify logout"},
		{name: "unknown page fails closed", page: "<html>proxy notice</html>", wantErr: "cannot verify logout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logoutVerdict(tt.page)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("logoutVerdict(%q) = %v, want success", tt.page, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("logoutVerdict(%q) error = %v, want containing %q", tt.page, err, tt.wantErr)
			}
		})
	}
}

func TestLoginVerdict(t *testing.T) {
	tests := []struct {
		name    string
		page    string
		wantErr string // "" means success
	}{
		{name: "session page after login is success", page: "usted ya está conectado", wantErr: ""},
		{name: "session marker ya conectado is success", page: "usuario ya conectado", wantErr: ""},
		{name: "login page means rejected", page: `<form action="LoginServlet">Bienvenido</form>`, wantErr: "portal still requires authentication"},
		{name: "login marker only means rejected", page: "LoginServlet", wantErr: "portal still requires authentication"},
		{name: "empty page fails closed", page: "", wantErr: "portal returned an unrecognized page"},
		{name: "unknown page fails closed", page: "<html>proxy notice</html>", wantErr: "portal returned an unrecognized page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loginVerdict(tt.page)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("loginVerdict(%q) = %v, want success", tt.page, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("loginVerdict(%q) error = %v, want containing %q", tt.page, err, tt.wantErr)
			}
		})
	}
}

func TestVerifyLogin(t *testing.T) {
	tests := []struct {
		name    string
		fetch   func() (string, error)
		wantErr string // "" means success
	}{
		{name: "connected page is success", fetch: func() (string, error) { return "usted ya está conectado", nil }, wantErr: ""},
		{name: "needs auth page is rejected", fetch: func() (string, error) { return `<form action="LoginServlet">Bienvenido</form>`, nil }, wantErr: "portal still requires authentication"},
		{name: "unknown page fails closed", fetch: func() (string, error) { return "<html>proxy notice</html>", nil }, wantErr: "portal returned an unrecognized page"},
		{name: "fetch error fails closed", fetch: func() (string, error) { return "", errors.New("boom") }, wantErr: "cannot verify login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyLogin(tt.fetch)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("verifyLogin = %v, want success", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("verifyLogin error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
