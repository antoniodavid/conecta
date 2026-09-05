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

func TestParseIWInterfacesSkipsP2PDevice(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "real iw dev with unnamed P2P block yields wlo1",
			input: "phy#0\n\tUnnamed/non-netdev interface\n\t\twdev 0xe\n\t\taddr 40:74:e0:a5:ee:89\n\t\ttype P2P-device\n\tInterface wlo1\n\t\tifindex 3\n\t\twdev 0x1\n\t\taddr 40:74:e0:a5:ee:89\n\t\ttype managed\n",
			want:  []string{"wlo1"},
		},
		{
			name:  "Interface P2P-device block skipped",
			input: "phy#0\n\tInterface p2p-dev-wlo1\n\t\tifindex 4\n\t\ttype P2P-device\n\tInterface wlo1\n\t\tifindex 3\n\t\ttype managed\n",
			want:  []string{"wlo1"},
		},
		{
			name:  "only P2P-device errors",
			input: "phy#0\n\tInterface p2p-dev-wlo1\n\t\tifindex 4\n\t\ttype P2P-device\n",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIWInterfaces(tt.input)
			if tt.want == nil {
				if err == nil {
					t.Fatalf("P2P-only output must return an error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseIWInterfaces error = %v", err)
			}
			if len(got) != len(tt.want) || got[0] != tt.want[0] {
				t.Fatalf("ParseIWInterfaces = %v, want %v", got, tt.want)
			}
		})
	}
}
