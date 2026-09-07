package network

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRouteStub installs fake `ip` and `sudo` on PATH (prepended, repo
// pattern). The ip stub logs every invocation and reports the portal route
// as present when STUB_ROUTE_PRESENT=1, or after a sudo add attempt when
// STUB_RACE=1 (the route "appears" between the failed add and the re-check).
// The sudo stub logs, marks STUB_ADDED, and exits STUB_SUDO_ADD_EXIT (0 by
// default). Nothing touches the live host.
func writeRouteStub(t *testing.T, present, race, sudoAddExit string) (dir, log string) {
	t.Helper()
	dir = t.TempDir()
	log = filepath.Join(dir, "calls.log")
	added := filepath.Join(dir, "added")

	ip := "#!/bin/sh\necho \"ip $*\" >> \"$STUB_LOG\"\n" +
		"case \"$*\" in\n" +
		"  *\"route show 10.180.0.0/16\"*)\n" +
		"    if [ \"$STUB_ROUTE_PRESENT\" = \"1\" ] || { [ -f \"$STUB_ADDED\" ] && [ \"$STUB_RACE\" = \"1\" ]; }; then\n" +
		"      echo \"10.180.0.0/16 via 192.168.1.1 dev enp3s0\"\n" +
		"      exit 0\n" +
		"    fi\n" +
		"    exit 1;;\n" +
		"esac\nexit 0\n"
	sudo := "#!/bin/sh\necho \"sudo $*\" >> \"$STUB_LOG\"\n" +
		"case \"$*\" in\n" +
		"  *\"ip route add 10.180.0.0/16 via\"*)\n" +
		"    touch \"$STUB_ADDED\"\n" +
		"    echo \"stub: sudo ip route add failed\"\n" +
		"    exit \"${STUB_SUDO_ADD_EXIT:-0}\";;\n" +
		"esac\nexit 0\n"
	for name, body := range map[string]string{"ip": ip, "sudo": sudo} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0755); err != nil {
			t.Fatalf("write stub %s: %v", name, err)
		}
	}
	t.Setenv("STUB_LOG", log)
	t.Setenv("STUB_ADDED", added)
	t.Setenv("STUB_ROUTE_PRESENT", present)
	t.Setenv("STUB_RACE", race)
	t.Setenv("STUB_SUDO_ADD_EXIT", sudoAddExit)
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return dir, log
}

func routeLogLines(t *testing.T, log string) []string {
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

func countContains(lines []string, substr string) int {
	n := 0
	for _, l := range lines {
		if strings.Contains(l, substr) {
			n++
		}
	}
	return n
}

// Route already present: nil and no sudo call at all.
func TestEnsurePortalRouteAlreadyPresentNoSudo(t *testing.T) {
	_, log := writeRouteStub(t, "1", "", "")
	if err := EnsurePortalRoute("192.168.1.1", "enp3s0"); err != nil {
		t.Fatalf("EnsurePortalRoute error = %v, want nil", err)
	}
	lines := routeLogLines(t, log)
	if countContains(lines, "sudo ") != 0 {
		t.Fatalf("present route must not invoke sudo, log = %v", lines)
	}
	if countContains(lines, "ip route show 10.180.0.0/16") != 1 {
		t.Fatalf("expected one read-only check, log = %v", lines)
	}
}

// Absent route: sudo add runs with the exact reference argv (no shell).
func TestEnsurePortalRouteAddsWithExactArgv(t *testing.T) {
	_, log := writeRouteStub(t, "", "", "0")
	if err := EnsurePortalRoute("192.168.1.1", "enp3s0"); err != nil {
		t.Fatalf("EnsurePortalRoute error = %v, want nil", err)
	}
	lines := routeLogLines(t, log)
	if countContains(lines, "sudo -n ip route add 10.180.0.0/16 via 192.168.1.1 dev enp3s0") != 1 {
		t.Fatalf("expected one fixed-argv sudo add, log = %v", lines)
	}
	if countContains(lines, "ip route show") != 1 {
		t.Fatalf("absent route must check once before the add, log = %v", lines)
	}
}

// sudo add fails but the route then shows up (race): nil.
func TestEnsurePortalRouteAddFailsButRouteAppears(t *testing.T) {
	_, log := writeRouteStub(t, "", "1", "1")
	if err := EnsurePortalRoute("192.168.1.1", "enp3s0"); err != nil {
		t.Fatalf("EnsurePortalRoute error = %v, want nil when the re-check sees the route", err)
	}
	lines := routeLogLines(t, log)
	if countContains(lines, "ip route show") != 2 {
		t.Fatalf("failed add must trigger one re-check, log = %v", lines)
	}
}

// sudo add fails and the route is still absent: wrapped error with output.
func TestEnsurePortalRouteAddFailsStillAbsent(t *testing.T) {
	_, log := writeRouteStub(t, "", "", "1")
	err := EnsurePortalRoute("192.168.1.1", "enp3s0")
	if err == nil {
		t.Fatal("EnsurePortalRoute = nil, want error when the route stays absent")
	}
	for _, want := range []string{"failed to add portal route", "stub: sudo ip route add failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error must carry %q, got %q", want, err.Error())
		}
	}
	lines := routeLogLines(t, log)
	if countContains(lines, "sudo -n ip route add") != 1 {
		t.Fatalf("expected exactly one sudo add attempt, log = %v", lines)
	}
}

// Empty gateway/iface fails fast, before any exec.
func TestEnsurePortalRouteEmptyArgs(t *testing.T) {
	for _, tt := range []struct {
		name    string
		gateway string
		iface   string
	}{
		{"both empty", "", ""},
		{"empty gateway", "", "enp3s0"},
		{"empty iface", "192.168.1.1", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := writeRouteStub(t, "", "", "0")
			if err := EnsurePortalRoute(tt.gateway, tt.iface); err == nil {
				t.Fatal("EnsurePortalRoute = nil, want error on empty gateway/iface")
			}
			if lines := routeLogLines(t, log); len(lines) != 0 {
				t.Fatalf("empty args must not exec anything, log = %v", lines)
			}
		})
	}
}
