package config

import (
	"os"
	"path/filepath"
	"testing"
)

// RED 5.1: saved configs may contain credentials and must be owner-only.
func TestSaveRestrictsPermissions(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := DefaultConfig()
	cfg.Credentials.Username = "user"
	cfg.Credentials.Password = "secret"

	if err := Save(cfg, cfgPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	fi, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Fatalf("saved config perm = %04o, want 0600", perm)
	}
}
