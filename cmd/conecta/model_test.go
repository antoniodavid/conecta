package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"conecta/pkg/config"
	"conecta/pkg/hotspot"
)

// RED 2.4: hotspot status refresh must be observable through Update.
func TestUpdateHotspotStatusMsgReflectsResult(t *testing.T) {
	m := NewModel(config.DefaultConfig())
	updated, _ := m.Update(hotspotStatusMsg{
		status: &hotspot.Status{Active: true, SSID: "TEST", Clients: 3},
	})
	got := updated.(Model)
	if got.hotspotStatus == nil || !got.hotspotStatus.Active || got.hotspotStatus.Clients != 3 {
		t.Fatalf("hotspotStatusMsg must refresh displayed state, got %+v", got.hotspotStatus)
	}
}

// RED 2.4: transient messages must expire even when a key message arrives first.
func TestUpdateKeyMsgExpiresStaleMessage(t *testing.T) {
	m := NewModel(config.DefaultConfig())
	m.statusMsg = "stale"
	m.statusExp = time.Now().Add(-time.Second)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if got := updated.(Model); got.statusMsg != "" {
		t.Fatalf("stale status message must expire on key input, kept %q", got.statusMsg)
	}
}

// RED 2.4: real auto-login command exists; nil when no credentials are configured.
func TestAutoLoginNilWithoutCredentials(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Credentials.Username = ""
	cfg.Credentials.Password = ""
	m := NewModel(cfg)
	if cmd := m.autoLogin(); cmd != nil {
		t.Fatalf("autoLogin without credentials must be nil, got %v", cmd)
	}
}

func TestAutoLoginCmdWithCredentials(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Credentials.Username = "user"
	cfg.Credentials.Password = "pass"
	m := NewModel(cfg)
	if cmd := m.autoLogin(); cmd == nil {
		t.Fatalf("autoLogin with credentials must return a command")
	}
}

// RED 2.4: hotspot check command produces a hotspotStatusMsg.
func TestCheckHotspotCmdProducesMsg(t *testing.T) {
	m := NewModel(config.DefaultConfig())
	msg := m.checkHotspot()()
	if _, ok := msg.(hotspotStatusMsg); !ok {
		t.Fatalf("checkHotspot must produce hotspotStatusMsg, got %T", msg)
	}
}

// Guard: action results still set a future expiry.
func TestActionResultSetsExpiry(t *testing.T) {
	m := NewModel(config.DefaultConfig())
	updated, _ := m.Update(actionResultMsg{success: true, message: "ok"})
	got := updated.(Model)
	if got.statusMsg != "ok" || !got.statusExp.After(time.Now()) {
		t.Fatalf("actionResultMsg must set message with future expiry: %+v", got)
	}
}
