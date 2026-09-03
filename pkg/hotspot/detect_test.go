package hotspot

import (
	"testing"
)

// RED 2.1: same-line "Interface <name>" parse.
func TestParseIWInterfacesSameLine(t *testing.T) {
	out := "phy#0\n\tInterface wlan0\n\t\tifindex 3\n\t\ttype managed\nphy#1\n\tInterface ap0\n\t\tifindex 4\n"
	ifaces, err := ParseIWInterfaces(out)
	if err != nil {
		t.Fatalf("ParseIWInterfaces error = %v", err)
	}
	if len(ifaces) != 2 || ifaces[0] != "wlan0" || ifaces[1] != "ap0" {
		t.Fatalf("ParseIWInterfaces = %v, want [wlan0 ap0]", ifaces)
	}
}

func TestParseIWInterfacesHostileName(t *testing.T) {
	out := "phy#0\n\tInterface my wifi;$(evil)\n"
	ifaces, err := ParseIWInterfaces(out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ifaces) != 1 || ifaces[0] != "my wifi;$(evil)" {
		t.Fatalf("hostile interface name must be preserved verbatim: %v", ifaces)
	}
}

func TestParseIWInterfacesEmpty(t *testing.T) {
	if _, err := ParseIWInterfaces(""); err == nil {
		t.Fatalf("empty iw output must return an error")
	}
	if _, err := ParseIWInterfaces("phy#0\n\tno interfaces here\n"); err == nil {
		t.Fatalf("output without Interface lines must return an error")
	}
}
