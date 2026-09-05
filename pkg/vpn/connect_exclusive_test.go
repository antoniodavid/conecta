package vpn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fake-nmcli (+ip/sudo) stub dir prepended to PATH. The nmcli stub appends
// every invocation to $STUB_LOG and serves profile data from $NMCLI_CON_LIST
// ($NMCLI_ACTIVE for --active); `con down` fails when the target matches
// $NMCLI_DOWN_FAIL. ip/sudo stubs only log (sudo fails so the wg-quick
// fallback can never run live here).
func setupStub(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	nmcli := "#!/bin/sh\necho \"nmcli $*\" >> \"$STUB_LOG\"\n" +
		"case \"$*\" in\n" +
		"  *\"con show --active\"*) printf '%s' \"$NMCLI_ACTIVE\"; exit 0;;\n" +
		"  *\"con show\"*) printf '%s' \"$NMCLI_CON_LIST\"; exit 0;;\n" +
		"  *\"con down\"*)\n" +
		"    name=\"$3\"\n" +
		"    if [ -n \"$NMCLI_DOWN_FAIL\" ] && [ \"$NMCLI_DOWN_FAIL\" = \"$name\" ]; then echo \"down failed: $name\"; exit 1; fi\n" +
		"    exit 0;;\n" +
		"  *\"con up\"*) exit 0;;\n" +
		"esac\nexit 0\n"
	ip := "#!/bin/sh\necho \"ip $*\" >> \"$STUB_LOG\"\nexit 0\n"
	sudo := "#!/bin/sh\necho \"sudo $*\" >> \"$STUB_LOG\"\nexit 1\n"
	for name, body := range map[string]string{"nmcli": nmcli, "ip": ip, "sudo": sudo} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0755); err != nil {
			t.Fatalf("setup stub %s: %v", name, err)
		}
	}
	t.Setenv("STUB_LOG", log)
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return log
}

func readLog(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read log: %v", err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func indexOf(lines []string, substr string) int {
	for i, l := range lines {
		if strings.Contains(l, substr) {
			return i
		}
	}
	return -1
}

func containsLog(lines []string, substr string) bool { return indexOf(lines, substr) >= 0 }

// Downs of other actives come BEFORE up of target; inactive and non-wireguard
// profiles are never downed.
func TestConnectToExclusiveOrder(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\nExtra:11111111-1111-1111-1111-111111111111:wireguard\nWired connection 1:e843bf54-83c1-3612-ae7e-5b999df6f95f:802-3-ethernet\n")
	t.Setenv("NMCLI_ACTIVE", "USA:USA\nWired connection 1:eth0\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	if err := m.ConnectTo("Binhex-Spain"); err != nil {
		t.Fatalf("ConnectTo error = %v", err)
	}
	lines := readLog(t, log)
	downUSA := indexOf(lines, "nmcli con down USA")
	upTarget := indexOf(lines, "nmcli con up Binhex-Spain")
	if downUSA < 0 {
		t.Fatalf("expected down of other active USA, log = %v", lines)
	}
	if upTarget < 0 {
		t.Fatalf("expected up of target Binhex-Spain, log = %v", lines)
	}
	if downUSA > upTarget {
		t.Fatalf("down must precede up, log = %v", lines)
	}
	for _, wantAbsent := range []string{"nmcli con down Binhex-Spain", "nmcli con down Extra", "nmcli con down Wired"} {
		if containsLog(lines, wantAbsent) {
			t.Fatalf("must never down %q, log = %v", wantAbsent, lines)
		}
	}
	if containsLog(lines, "sudo ") {
		t.Fatalf("wg-quick fallback must not run when nmcli up succeeds, log = %v", lines)
	}
}

// A failing down aborts before up (fail closed, no dual-active risk).
func TestConnectToDownFailureAbortsBeforeUp(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "USA:USA\nBinhex-Spain:Binhex-Spain\n")
	t.Setenv("NMCLI_DOWN_FAIL", "USA")

	m := NewManager("wg0", "USA")
	if err := m.ConnectTo("Binhex-Spain"); err == nil {
		t.Fatalf("expected down failure to abort ConnectTo")
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con down USA") {
		t.Fatalf("expected attempted down of USA, log = %v", lines)
	}
	if containsLog(lines, "nmcli con up") {
		t.Fatalf("must not bring target up after down failure, log = %v", lines)
	}
}

// Unknown name errors without any up/down exec.
func TestConnectToUnknownName(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "USA:USA\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	err := m.ConnectTo("Nope")
	if err == nil {
		t.Fatalf("expected unknown profile error")
	}
	if !strings.Contains(err.Error(), `unknown VPN profile "Nope"`) || !strings.Contains(err.Error(), "available: USA, Binhex-Spain") {
		t.Fatalf("error must keep message shape listing available, got %q", err.Error())
	}
	for _, l := range readLog(t, log) {
		if strings.Contains(l, "nmcli con up") || strings.Contains(l, "nmcli con down") {
			t.Fatalf("unknown name must not exec up/down, log line = %q", l)
		}
	}
}

// Single-profile case just ups with no downs.
func TestConnectToSingleProfile(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "USA")
	if err := m.ConnectTo("USA"); err != nil {
		t.Fatalf("ConnectTo error = %v", err)
	}
	lines := readLog(t, log)
	if !containsLog(lines, "nmcli con up USA") {
		t.Fatalf("expected up of single profile, log = %v", lines)
	}
	if containsLog(lines, "nmcli con down") {
		t.Fatalf("single profile must not down anything, log = %v", lines)
	}
}

// Connect uses the configured name with the same exclusivity.
func TestConnectUsesConfiguredNameExclusively(t *testing.T) {
	log := setupStub(t)
	t.Setenv("NMCLI_CON_LIST", "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\nBinhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n")
	t.Setenv("NMCLI_ACTIVE", "USA:USA\n")
	t.Setenv("NMCLI_DOWN_FAIL", "")

	m := NewManager("wg0", "Binhex-Spain")
	if err := m.Connect(); err != nil {
		t.Fatalf("Connect error = %v", err)
	}
	lines := readLog(t, log)
	downUSA := indexOf(lines, "nmcli con down USA")
	upTarget := indexOf(lines, "nmcli con up Binhex-Spain")
	if downUSA < 0 || upTarget < 0 || downUSA > upTarget {
		t.Fatalf("Connect must down other actives before up of configured name, log = %v", lines)
	}
}
