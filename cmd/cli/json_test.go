package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// RED 1.1: envelope contract — one JSON line on stdout, diagnostics on stderr, exits 0/2/3/4.
func TestEnvelopeSuccessSingleLine(t *testing.T) {
	out := buildResult(map[string]string{"status": "connected"})
	if strings.Contains(strings.TrimSpace(out), "\n") {
		t.Fatalf("envelope must be a single line, got %q", out)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
		t.Fatalf("stdout must be parseable JSON: %v (out=%q)", err, out)
	}
	if !env.Ok {
		t.Fatalf("success envelope ok must be true: %q", out)
	}
	if len(env.Data) == 0 {
		t.Fatalf("success envelope must carry data: %q", out)
	}
	if env.Error != nil {
		t.Fatalf("success envelope must not carry error: %q", out)
	}
}

func TestEnvelopeFailureCodes(t *testing.T) {
	cases := []struct {
		name string
		code string
		exit int
	}{
		{"invalid input", "invalid_input", 2},
		{"config", "config", 2},
		{"unavailable", "unavailable", 3},
		{"op failed", "op_failed", 3},
		{"authz", "authz", 4},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out, exit := buildError(tt.code, "boom")
			if exit != tt.exit {
				t.Fatalf("buildError(%q) exit = %d, want %d", tt.code, exit, tt.exit)
			}
			var env Envelope
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
				t.Fatalf("failure stdout must be parseable JSON: %v", err)
			}
			if env.Ok {
				t.Fatalf("failure envelope ok must be false: %q", out)
			}
			if env.Error == nil || env.Error.Code != tt.code {
				t.Fatalf("failure envelope code must be %q: %q", tt.code, out)
			}
			if strings.Contains(strings.TrimSpace(out), "\n") {
				t.Fatalf("failure envelope must be a single line: %q", out)
			}
		})
	}
}

func TestEnvelopeExitMapping(t *testing.T) {
	if exitForCode("invalid_input") != 2 {
		t.Fatalf("invalid_input must map to exit 2")
	}
	if exitForCode("config") != 2 {
		t.Fatalf("config must map to exit 2")
	}
	if exitForCode("unavailable") != 3 {
		t.Fatalf("unavailable must map to exit 3")
	}
	if exitForCode("op_failed") != 3 {
		t.Fatalf("op_failed must map to exit 3")
	}
	if exitForCode("authz") != 4 {
		t.Fatalf("authz must map to exit 4")
	}
}

// Login flags bind to the login subcommand only (per-subcommand FlagSets).
// Note: rejectSubcommandFlags itself calls emitError (os.Exit), so the
// non-login side asserts hasLoginFlags — the exact predicate production
// uses — rather than invoking the exiting path.
func TestLoginFlagsBindToLoginOnly(t *testing.T) {
	t.Run("login binds user/pass flags", func(t *testing.T) {
		cases := []struct {
			name     string
			args     []string
			wantUser string
			wantPass string
			wantErr  bool
			wantRest int
		}{
			{"space separated", []string{"--user", "alice", "--pass", "s3cret"}, "alice", "s3cret", false, 0},
			{"equals form", []string{"--user=bob", "--pass=hunter2"}, "bob", "hunter2", false, 0},
			{"no flags falls back to config", nil, "", "", false, 0},
			{"unknown flag rejected", []string{"--token", "x"}, "", "", true, 0},
			{"missing value rejected", []string{"--user"}, "", "", true, 0},
			{"positionals surfaced as rest", []string{"--user", "alice", "extra"}, "alice", "", false, 1},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				user, pass, rest, err := parseLoginFlags(tt.args)
				if tt.wantErr {
					if err == nil {
						t.Fatalf("parseLoginFlags(%v) err = nil, want error", tt.args)
					}
					return
				}
				if err != nil {
					t.Fatalf("parseLoginFlags(%v) err = %v, want nil", tt.args, err)
				}
				if user != tt.wantUser || pass != tt.wantPass {
					t.Fatalf("parseLoginFlags(%v) = (%q, %q), want (%q, %q)", tt.args, user, pass, tt.wantUser, tt.wantPass)
				}
				if len(rest) != tt.wantRest {
					t.Fatalf("parseLoginFlags(%v) rest = %v, want %d leftover", tt.args, rest, tt.wantRest)
				}
			})
		}
	})

	t.Run("other subcommands reject login flags", func(t *testing.T) {
		cases := []struct {
			name         string
			args         []string
			wantRejected bool
		}{
			{"status clean", []string{}, false},
			{"status with user flag", []string{"--user", "alice"}, true},
			{"status with user equals", []string{"--user=alice"}, true},
			{"logout with pass flag", []string{"--pass", "s3cret"}, true},
			{"speed with pass equals", []string{"--pass=s3cret"}, true},
			{"similar prefix not a login flag", []string{"--username", "alice"}, false},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				if got := hasLoginFlags(tt.args); got != tt.wantRejected {
					t.Fatalf("hasLoginFlags(%v) = %v, want %v", tt.args, got, tt.wantRejected)
				}
			})
		}
	})
}
