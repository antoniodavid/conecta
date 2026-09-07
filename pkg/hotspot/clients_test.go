package hotspot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLeaseFixture creates a create_ap-style temp dir (any wireless iface
// prefix, e.g. wlo1) under $TMPDIR with the given dnsmasq lease lines.
func writeLeaseFixture(t *testing.T, lines ...string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp) // os.TempDir() honors TMPDIR on unix
	dir := filepath.Join(tmp, "create_ap.wlo1.conf.Iw1CxEfH")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := strings.Join(lines, "\n")
	if err := os.WriteFile(filepath.Join(dir, "dnsmasq.leases"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return tmp
}

// writePATHStub installs one executable script into a fresh temp dir that is
// PREPENDED to $PATH (multiple stubs coexist; first match wins).
func writePATHStub(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestParseLeaseLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantOK   bool
		wantIP   string
		wantMAC  string
		wantName string
	}{
		{"real lease line", "1723030097 22:df:51:de:e8:40 192.168.12.129 Xiaomi-11-Lite-5G-NE", true, "192.168.12.129", "22:DF:51:DE:E8:40", "Xiaomi-11-Lite-5G-NE"},
		{"stock dnsmasq order (expiry ip mac name)", "1723030097 192.168.12.129 22:df:51:de:e8:40 Xiaomi-11-Lite-5G-NE", true, "192.168.12.129", "22:DF:51:DE:E8:40", "Xiaomi-11-Lite-5G-NE"},
		{"star name becomes empty", "1723030097 192.168.12.5 aa:bb:cc:dd:ee:01 *", true, "192.168.12.5", "AA:BB:CC:DD:EE:01", ""},
		{"mac and ip both present", "1723030097 aa:bb:cc:dd:ee:01 192.168.12.5 host", true, "192.168.12.5", "AA:BB:CC:DD:EE:01", "host"},
		{"too few fields", "1723030097 192.168.12.129", false, "", "", ""},
		{"empty line", "", false, "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, ok := parseLeaseLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("parseLeaseLine(%q) ok = %v, want %v", tt.line, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if l.ip != tt.wantIP || l.mac != tt.wantMAC || l.name != tt.wantName {
				t.Fatalf("parseLeaseLine(%q) = %+v, want ip=%q mac=%q name=%q", tt.line, l, tt.wantIP, tt.wantMAC, tt.wantName)
			}
		})
	}
}

func TestReadDHCPLeasesWlo1StyleDir(t *testing.T) {
	writeLeaseFixture(t,
		"1723030097 22:df:51:de:e8:40 192.168.12.129 Xiaomi-11-Lite-5G-NE",
		"1723030097 aa:bb:cc:dd:ee:01 192.168.12.5 *",
		"garbage",
	)
	cm := NewClientManager("ap0")
	leases := cm.readDHCPLeases()
	if len(leases) != 2 {
		t.Fatalf("readDHCPLeases = %d leases, want 2 (wlo1-style dir must be found via TMPDIR)", len(leases))
	}
	if leases[0].mac != "22:DF:51:DE:E8:40" || leases[0].ip != "192.168.12.129" || leases[0].name != "Xiaomi-11-Lite-5G-NE" {
		t.Fatalf("lease[0] = %+v", leases[0])
	}
	if leases[1].name != "" {
		t.Fatalf("star name must map to empty, got %q", leases[1].name)
	}
}

func TestLeasesToClients(t *testing.T) {
	leases := []dhcpLease{
		{ip: "192.168.12.129", mac: "22:DF:51:DE:E8:40", name: "Xiaomi"},
		{ip: "192.168.12.5", mac: "AA:BB:CC:DD:EE:01", name: ""},
	}
	clients := leasesToClients(leases)
	if len(clients) != 2 {
		t.Fatalf("leasesToClients = %d clients, want 2", len(clients))
	}
	if clients[0].MAC != "22:DF:51:DE:E8:40" || clients[0].IP != "192.168.12.129" || clients[0].Name != "Xiaomi" {
		t.Fatalf("clients[0] = %+v", clients[0])
	}
	if clients[1].Name != "" {
		t.Fatalf("clients[1].Name = %q, want empty", clients[1].Name)
	}
}

// TestListClientsFromLeasesWhenIWEmpty: the live driver returns an empty
// `iw station dump` even with a client attached, so the lease file must be
// the fallback source and ARP enrichment must still run (no panic).
func TestListClientsFromLeasesWhenIWEmpty(t *testing.T) {
	writeLeaseFixture(t, "1723030097 22:df:51:de:e8:40 192.168.12.129 Xiaomi-11-Lite-5G-NE")
	writePATHStub(t, "iw", "#!/bin/sh\nexit 0\n") // station dump yields nothing, rc 0
	writePATHStub(t, "ip", "#!/bin/sh\nif [ \"$1\" = \"neigh\" ]; then echo '192.168.12.129 dev ap0 lladdr 22:df:51:de:e8:40 REACHABLE'; fi\nexit 0\n")

	cm := NewClientManager("ap0")
	clients, err := cm.ListClients()
	if err != nil {
		t.Fatalf("ListClients error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("ListClients = %d clients, want 1 from leases when iw is empty", len(clients))
	}
	c := clients[0]
	if c.MAC != "22:DF:51:DE:E8:40" || c.IP != "192.168.12.129" || c.Name != "Xiaomi-11-Lite-5G-NE" {
		t.Fatalf("client = %+v, want lease values", c)
	}
}

// TestListClientsARPFillsMissingIP: iw reports a station (MAC only) and ARP
// provides the IP; enrichment must run and fill it. TMPDIR is isolated so a
// live /tmp/create_ap.* lease on this host cannot leak into the result.
func TestListClientsARPFillsMissingIP(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir()) // no lease fixture: pure iw + ARP path
	writePATHStub(t, "iw", "#!/bin/sh\nif [ \"$3\" = \"station\" ]; then printf 'Station 22:df:51:de:e8:40 (on ap0)\n\tsignal: -45 dBm\n\tRX bytes: 100\n\tTX bytes: 50\n'; fi\nexit 0\n")
	writePATHStub(t, "ip", "#!/bin/sh\nif [ \"$1\" = \"neigh\" ]; then echo '192.168.12.129 dev ap0 lladdr 22:df:51:de:e8:40 REACHABLE'; fi\nexit 0\n")

	cm := NewClientManager("ap0")
	clients, err := cm.ListClients()
	if err != nil {
		t.Fatalf("ListClients error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("ListClients = %d clients, want 1", len(clients))
	}
	if clients[0].MAC != "22:DF:51:DE:E8:40" || clients[0].IP != "192.168.12.129" {
		t.Fatalf("client = %+v, want MAC + ARP IP", clients[0])
	}
	if clients[0].RXBytes != 100 || clients[0].TXBytes != 50 {
		t.Fatalf("iw byte counters not kept: %+v", clients[0])
	}
}

func TestKickClient(t *testing.T) {
	const mac = "22:DF:51:DE:E8:40"

	// sudo stub records argv and honors STUB_KICK_EXIT for the error path.
	sudoStub := "#!/bin/sh\nprintf '%s' \"$*\" > \"$STUB_LOG\"\n[ \"${STUB_KICK_EXIT:-0}\" = \"0\" ] || { echo 'stub deny' >&2; exit \"${STUB_KICK_EXIT}\"; }\nexit 0\n"
	// Fresh log file per subtest so stale writes cannot leak between cases.
	setupKick := func(t *testing.T) string {
		t.Helper()
		// iw stub yields no AP interface, so empty-iface resolution falls
		// back to ap0 without ever invoking the real iw.
		writePATHStub(t, "iw", "#!/bin/sh\nexit 0\n")
		writePATHStub(t, "sudo", sudoStub)
		logPath := filepath.Join(t.TempDir(), "args.log")
		t.Setenv("STUB_LOG", logPath)
		return logPath
	}

	t.Run("argv uses fallback ap0 when iface empty", func(t *testing.T) {
		logPath := setupKick(t)
		if err := KickClient("", mac); err != nil {
			t.Fatalf("KickClient error = %v", err)
		}
		got, _ := os.ReadFile(logPath)
		want := "-n iw dev ap0 station del " + mac + " subtype 0xC"
		if string(got) != want {
			t.Fatalf("sudo argv = %q, want %q", got, want)
		}
	})

	t.Run("provided iface used verbatim", func(t *testing.T) {
		logPath := setupKick(t)
		if err := KickClient("wlan9", mac); err != nil {
			t.Fatalf("KickClient error = %v", err)
		}
		got, _ := os.ReadFile(logPath)
		if !strings.Contains(string(got), "dev wlan9 station del") {
			t.Fatalf("sudo argv = %q, want wlan9 iface", got)
		}
	})

	t.Run("invalid MAC rejected before sudo", func(t *testing.T) {
		logPath := setupKick(t)
		err := KickClient("ap0", "not-a-mac")
		if !errors.Is(err, ErrInvalidMAC) {
			t.Fatalf("KickClient invalid MAC error = %v, want ErrInvalidMAC", err)
		}
		if b, _ := os.ReadFile(logPath); len(b) != 0 {
			t.Fatalf("sudo must not run for invalid MAC, logged %q", b)
		}
	})

	t.Run("sudo failure wraps output", func(t *testing.T) {
		setupKick(t)
		t.Setenv("STUB_KICK_EXIT", "1")
		err := KickClient("ap0", mac)
		if err == nil {
			t.Fatal("KickClient must error when sudo fails")
		}
		if !strings.Contains(err.Error(), "stub deny") {
			t.Fatalf("error must wrap sudo output, got %v", err)
		}
	})
}

func TestValidMAC(t *testing.T) {
	valid := []string{"aa:bb:cc:dd:ee:ff", "22:DF:51:DE:E8:40", "00:11:22:33:44:55"}
	for _, m := range valid {
		if !validMAC(m) {
			t.Errorf("validMAC(%q) = false, want true", m)
		}
	}
	invalid := []string{"", "aa:bb:cc", "gg:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:f", "aa:bb:cc:dd:ee:ff:00", "aabbccddeeff"}
	for _, m := range invalid {
		if validMAC(m) {
			t.Errorf("validMAC(%q) = true, want false", m)
		}
	}
}
