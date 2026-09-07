package hotspot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCmdStub installs fake `sudo` and `iw` on PATH. Every privileged
// hotspot command goes through `sudo`, so the stub logs each invocation and
// picks the exit code per command from env vars; the fake `iw` returns a fixed
// interface so writeConfig succeeds without touching the live system.
func writeCmdStub(t *testing.T, authzExit, startExit, stopExit string) string {
	t.Helper()
	dir := t.TempDir()

	sudo := filepath.Join(dir, "sudo")
	sudoScript := `#!/bin/sh
echo "$*" >> "$STUB_LOG"
case "$*" in
  *"systemctl is-active create_ap"*) exit "${STUB_AUTHZ_EXIT:-0}" ;;
  *"systemctl start create_ap"*) exit "${STUB_START_EXIT:-0}" ;;
  *"systemctl stop create_ap"*) exit "${STUB_STOP_EXIT:-0}" ;;
esac
exit 0
`
	if err := os.WriteFile(sudo, []byte(sudoScript), 0o755); err != nil {
		t.Fatal(err)
	}

	iw := filepath.Join(dir, "iw")
	iwScript := "#!/bin/sh\nprintf 'Interface wlan0\\n\\ttype managed\\n'\n"
	if err := os.WriteFile(iw, []byte(iwScript), 0o755); err != nil {
		t.Fatal(err)
	}

	log := filepath.Join(dir, "cmds.log")
	t.Setenv("PATH", dir)
	t.Setenv("STUB_LOG", log)
	t.Setenv("STUB_AUTHZ_EXIT", authzExit)
	t.Setenv("STUB_START_EXIT", startExit)
	t.Setenv("STUB_STOP_EXIT", stopExit)
	return log
}

func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestStartRestoresRadioOnServiceFailure(t *testing.T) {
	log := writeCmdStub(t, "3", "1", "0")
	if err := NewCreateAP(DefaultConfig()).Start(); err == nil {
		t.Fatal("Start() = nil error, want error when create_ap start fails")
	}
	if got := readLog(t, log); !strings.Contains(got, "nmcli r wifi on") {
		t.Fatalf("Start() failure did not restore radio; commands:\n%s", got)
	}
}

func TestStartDoesNotRestoreRadioOnSuccess(t *testing.T) {
	log := writeCmdStub(t, "3", "0", "0")
	if err := NewCreateAP(DefaultConfig()).Start(); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if got := readLog(t, log); strings.Contains(got, "nmcli r wifi on") {
		t.Fatalf("Start() success restored radio; commands:\n%s", got)
	}
}

func TestStopAlwaysRestoresRadio(t *testing.T) {
	for _, stopExit := range []string{"0", "1"} {
		log := writeCmdStub(t, "3", "0", stopExit)
		err := NewCreateAP(DefaultConfig()).Stop()
		if stopExit == "1" && err == nil {
			t.Fatal("Stop() = nil error, want error when systemctl stop fails")
		}
		if stopExit == "0" && err != nil {
			t.Fatalf("Stop() unexpected error: %v", err)
		}
		if got := readLog(t, log); !strings.Contains(got, "nmcli r wifi on") {
			t.Fatalf("Stop() (stop exit %s) did not restore radio; commands:\n%s", stopExit, got)
		}
	}
}