package hotspot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSudoStub installs a fake `sudo` on PATH that exits with
// STUB_SUDO_EXIT when invoked for the authz probe.
func writeSudoStub(t *testing.T, exitCode string) {
	t.Helper()
	dir := t.TempDir()
	stub := filepath.Join(dir, "sudo")
	script := "#!/bin/sh\ncase \"$*\" in\n  *\"systemctl is-active create_ap\"*) exit \"${STUB_SUDO_EXIT:-0}\" ;;\nesac\nexit 0\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("STUB_SUDO_EXIT", exitCode)
}

func TestCheckAuthz(t *testing.T) {
	tests := []struct {
		name      string
		exitCode  string
		wantError bool
	}{
		{"sudo allowed, unit active", "0", false},
		{"sudo allowed, unit inactive", "3", false},
		{"sudo denied", "1", true},
		{"sudo command not found", "126", true},
		{"sudo unavailable", "127", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeSudoStub(t, tt.exitCode)
			err := checkAuthz()
			if tt.wantError != (err != nil) {
				t.Fatalf("checkAuthz() error = %v, wantError = %v", err, tt.wantError)
			}
			if tt.wantError && !strings.Contains(err.Error(), "./deploy.sh --setup-privileges") {
				t.Fatalf("authz error %q missing setup-privileges guidance", err)
			}
		})
	}
}