package hotspot

import "testing"

// RED 3.1: same-line "Interface <name>" tracking with following "type AP"
// detection. Fixtures mirror the live host where ap0 is type AP while the
// create_ap unit may be inactive.
func TestParseAPInterface(t *testing.T) {
	realIWDev := "phy#0\n\tUnnamed/non-netdev interface\n\t\twdev 0xe\n\t\taddr 40:74:e0:a5:ee:89\n\t\ttype P2P-device\n\tInterface wlan0\n\t\tifindex 3\n\t\twdev 0x1\n\t\taddr 40:74:e0:a5:ee:89\n\t\ttype managed\n\tInterface ap0\n\t\tifindex 4\n\t\twdev 0x2\n\t\ttype AP\n"
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "real iw dev output yields ap0",
			input: realIWDev,
			want:  "ap0",
		},
		{
			name:  "skips P2P-device and unnamed blocks",
			input: "phy#0\n\tUnnamed/non-netdev interface\n\t\ttype P2P-device\n\tInterface p2p-dev-wlan0\n\t\ttype P2P-device\n\tInterface ap0\n\t\tifindex 4\n\t\ttype AP\n",
			want:  "ap0",
		},
		{
			name:  "multiple phys finds AP on second phy",
			input: "phy#0\n\tInterface wlan0\n\t\ttype managed\nphy#1\n\tInterface ap0\n\t\ttype AP\n",
			want:  "ap0",
		},
		{
			name:  "no AP interface yields empty",
			input: "phy#0\n\tInterface wlan0\n\t\ttype managed\n\tInterface wlan1\n\t\ttype managed\n",
			want:  "",
		},
		{
			name:  "empty output yields empty",
			input: "",
			want:  "",
		},
		{
			name:  "interface without type line yields empty",
			input: "phy#0\n\tInterface ap0\n\t\tifindex 4\n",
			want:  "",
		},
		{
			name:  "AP type is case-insensitive",
			input: "phy#0\n\tInterface ap0\n\t\ttype ap\n",
			want:  "ap0",
		},
		{
			name:  "hostile AP name preserved verbatim",
			input: "phy#0\n\tInterface my ap;$(evil)\n\t\ttype AP\n",
			want:  "my ap;$(evil)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAPInterface(tt.input); got != tt.want {
				t.Fatalf("parseAPInterface = %q, want %q", got, tt.want)
			}
		})
	}
}
