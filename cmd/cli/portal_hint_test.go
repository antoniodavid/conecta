package main

import (
	"errors"
	"strings"
	"testing"
)

// VPN-up + portal-unreachable composes the disconnect-VPN guidance.
// Pure helper: vpnConnected is injected, tests never exec nmcli.
func TestPortalVPNHint(t *testing.T) {
	portalErrs := []struct {
		name string
		err  error
	}{
		{"logout GET phase", errors.New(`cannot access portal: Get "https://secure.etecsa.net:8443/": dial tcp 10.180.0.30:8443: i/o timeout`)},
		{"login GET phase", errors.New(`portal unreachable: Get "https://secure.etecsa.net:8443/": dial tcp 10.180.0.30:8443: i/o timeout`)},
	}
	for _, tt := range portalErrs {
		t.Run(tt.name+" with VPN up hints disconnect", func(t *testing.T) {
			got := withPortalVPNHint(tt.err, true)
			if got == nil {
				t.Fatalf("hint must not swallow the error")
			}
			if !strings.Contains(got.Error(), "vpn disconnect") {
				t.Fatalf("VPN-up portal failure must hint disconnect, got %q", got.Error())
			}
			if !strings.Contains(got.Error(), tt.err.Error()) {
				t.Fatalf("hint must preserve the original message, got %q", got.Error())
			}
		})
		t.Run(tt.name+" with VPN down passes through", func(t *testing.T) {
			if got := withPortalVPNHint(tt.err, false); got != tt.err {
				t.Fatalf("VPN-down must pass the error unchanged, got %q", got)
			}
		})
	}

	t.Run("non-portal error with VPN up unchanged", func(t *testing.T) {
		err := errors.New("credenciales inválidas")
		if got := withPortalVPNHint(err, true); got != err {
			t.Fatalf("non-portal error must pass through unchanged, got %q", got)
		}
	})

	t.Run("nil stays nil", func(t *testing.T) {
		if got := withPortalVPNHint(nil, true); got != nil {
			t.Fatalf("nil error must stay nil, got %q", got)
		}
	})
}

// The VPN status source is an injectable variable so production reads the
// real manager while tests stub it (never exec nmcli).
func TestVPNStatusCheckInjectable(t *testing.T) {
	defer func(orig func(string, string) bool) { vpnStatusCheck = orig }(vpnStatusCheck)
	vpnStatusCheck = func(_, _ string) bool { return true }
	if !vpnStatusCheck("wg0", "USA") {
		t.Fatalf("injected vpnStatusCheck stub must be honored")
	}
}
