package vpn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateImportFile(t *testing.T) {
	dir := t.TempDir()

	if err := validateImportFile(filepath.Join(dir, "missing.conf")); err == nil {
		t.Fatalf("missing file must fail validation")
	}

	notConf := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notConf, []byte("hello\n"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := validateImportFile(notConf); err == nil {
		t.Fatalf("non-conf file must fail validation")
	}

	okConf := filepath.Join(dir, "usa.conf")
	conf := "[Interface]\nPrivateKey = x\nAddress = 10.0.0.2/32\n\n[Peer]\nPublicKey = y\nEndpoint = 1.2.3.4:51820\n"
	if err := os.WriteFile(okConf, []byte(conf), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := validateImportFile(okConf); err != nil {
		t.Fatalf("valid conf must pass, got %v", err)
	}

	if err := validateImportFile(dir); err == nil {
		t.Fatalf("directory must fail validation")
	}
}

func TestParseImportedName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single quotes", "Connection 'USA' (0544e4d0-3bf8-4ae4-873c-14fedd922c74) successfully added.", "USA"},
		{"double quotes", `Connection "Spain" successfully added.`, "Spain"},
		{"no quoted name", "Connection successfully added.", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseImportedName(tt.input); got != tt.want {
				t.Fatalf("parseImportedName = %q, want %q", got, tt.want)
			}
		})
	}
}
