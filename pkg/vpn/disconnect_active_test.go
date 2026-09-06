package vpn

import (
	"strings"
	"testing"
)

// Disconnect with no active profile is a no-op success (only list reads).
func TestDisconnectNoActiveProfile(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "Wired connection 1:eth0\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	if err := m.Disconnect(); err != nil {
		t.Fatalf("Disconnect error = %v, want nil no-op", err)
	}
	for _, l := range readLog(t, log) {
		if strings.Contains(l, "con down") || strings.Contains(l, "sudo ") {
			t.Fatalf("no active profile must not exec down/fallback, log line = %q", l)
		}
	}
}

// Disconnect downs the ACTIVE profile, not the configured one when they differ.
func TestDisconnectDownsActiveNotConfigured(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "Binhex-Spain:Binhex-Spain\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	// Configured name is USA; the active one is Binhex-Spain.
	m := NewManager("wg0", "USA")
	if err := m.Disconnect(); err != nil {
		t.Fatalf("Disconnect error = %v", err)
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con down Binhex-Spain") {
		t.Fatalf("expected down of active Binhex-Spain, log = %v", lines)
	}
	if containsLog(lines, "nmcli con down USA") {
		t.Fatalf("must not down inactive configured USA, log = %v", lines)
	}
	if containsLog(lines, "sudo ") {
		t.Fatalf("no fallback on successful nmcli down, log = %v", lines)
	}
}

// DisconnectTo on a non-active (but existing) profile is a nil no-op.
func TestDisconnectToNonActiveNoop(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "Binhex-Spain:Binhex-Spain\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	if err := m.DisconnectTo("USA"); err != nil {
		t.Fatalf("DisconnectTo(inactive) error = %v, want nil", err)
	}
	for _, l := range readLog(t, log) {
		if strings.Contains(l, "con down") || strings.Contains(l, "sudo ") {
			t.Fatalf("inactive profile must not exec down/fallback, log line = %q", l)
		}
	}
}

// DisconnectTo on an unknown name errors with the available: shape, no exec.
func TestDisconnectToUnknownName(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "USA:USA\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	err := m.DisconnectTo("Nope")
	if err == nil {
		t.Fatalf("expected unknown profile error")
	}
	if !strings.Contains(err.Error(), `unknown VPN profile "Nope"`) || !strings.Contains(err.Error(), "available: USA, Binhex-Spain") {
		t.Fatalf("error must keep message shape listing available, got %q", err.Error())
	}
	for _, l := range readLog(t, log) {
		if strings.Contains(l, "con down") || strings.Contains(l, "sudo ") {
			t.Fatalf("unknown name must not exec down/fallback, log line = %q", l)
		}
	}
}

// DisconnectTo on the active profile with a failing nmcli down, name !=
// configured: the nmcli error is surfaced, no wg-quick fallback.
func TestDisconnectToDownFailNonConfigured(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "Binhex-Spain:Binhex-Spain\n")
	t.Setenv("NMCLI_DOWN_FAIL", "Binhex-Spain")

	m := NewManager("wg0", "USA") // configured USA, downing Binhex-Spain
	err := m.DisconnectTo("Binhex-Spain")
	if err == nil {
		t.Fatalf("expected nmcli down failure to surface")
	}
	if !strings.Contains(err.Error(), "Binhex-Spain") {
		t.Fatalf("error must name the failed profile, got %q", err.Error())
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con down Binhex-Spain") {
		t.Fatalf("expected attempted down of Binhex-Spain, log = %v", lines)
	}
	if containsLog(lines, "sudo ") {
		t.Fatalf("no wg-quick fallback for non-configured profile, log = %v", lines)
	}
}

// DisconnectTo on the configured active profile with a failing nmcli down
// attempts the wg-quick fallback.
func TestDisconnectToDownFailConfiguredFallsBack(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "USA:USA\n")
	t.Setenv("NMCLI_DOWN_FAIL", "USA")

	m := NewManager("wg0", "USA")
	err := m.DisconnectTo("USA")
	if err == nil {
		t.Fatalf("expected disconnect failure (sudo stub fails)")
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con down USA") {
		t.Fatalf("expected attempted nmcli down of USA, log = %v", lines)
	}
	if !containsLog(lines, "sudo wg-quick down wg0") {
		t.Fatalf("expected wg-quick fallback for configured profile, log = %v", lines)
	}
}

// Toggle with an active non-configured profile downs it and does NOT up
// anything.
func TestToggleActiveDisconnectsOnly(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "Binhex-Spain:Binhex-Spain\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	connected, err := m.Toggle()
	if err != nil {
		t.Fatalf("Toggle error = %v", err)
	}
	if connected {
		t.Fatalf("Toggle on active must report false")
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con down Binhex-Spain") {
		t.Fatalf("expected down of active Binhex-Spain, log = %v", lines)
	}
	if containsLog(lines, "nmcli con up") {
		t.Fatalf("toggle must not up anything when active, log = %v", lines)
	}
}

// Toggle with nothing active connects the configured profile exclusively.
func TestToggleNoneActiveConnectsConfigured(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	connected, err := m.Toggle()
	if err != nil {
		t.Fatalf("Toggle error = %v", err)
	}
	if !connected {
		t.Fatalf("Toggle with nothing active must report true")
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con up USA") {
		t.Fatalf("expected up of configured USA, log = %v", lines)
	}
	if containsLog(lines, "nmcli con down") {
		t.Fatalf("nothing active must not down anything, log = %v", lines)
	}
}