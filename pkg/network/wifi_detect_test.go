package network

import (
	"reflect"
	"testing"
)

// Table-driven same-line Interface parse incl. P2P skip.
// Fixtures include the real host `iw dev` output (USA/wlo1 host).
func TestParseWiFiInterfaces(t *testing.T) {
	realIWDev := "phy#0\n\tUnnamed/non-netdev interface\n\t\twdev 0xe\n\t\taddr 40:74:e0:a5:ee:89\n\t\ttype P2P-device\n\tInterface wlo1\n\t\tifindex 3\n\t\twdev 0x1\n\t\taddr 40:74:e0:a5:ee:89\n\t\ttype managed\n\t\tchannel 6 (2437 MHz), width: 20 MHz (no HT), center1: 2437 MHz\n\t\ttxpower 22.00 dBm\n"
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "real iw dev output yields wlo1",
			input: realIWDev,
			want:  []string{"wlo1"},
		},
		{
			name: "skips P2P-device Interface block",
			input: "phy#0\n\tInterface p2p-dev-wlo1\n\t\tifindex 4\n\t\twdev 0xe\n\t\ttype P2P-device\n\tInterface wlo1\n\t\tifindex 3\n\t\twdev 0x1\n\t\ttype managed\n",
			want: []string{"wlo1"},
		},
		{
			name: "only P2P-device yields nothing",
			input: "phy#0\n\tInterface p2p-dev-wlo1\n\t\tifindex 4\n\t\ttype P2P-device\n",
			want: nil,
		},
		{
			name: "accepts managed and type-less blocks",
			input: "phy#0\n\tInterface wlan0\n\t\tifindex 3\n\t\ttype managed\nphy#1\n\tInterface ap0\n\t\tifindex 4\n",
			want: []string{"wlan0", "ap0"},
		},
		{
			name: "accepts non-P2P types",
			input: "phy#0\n\tInterface wlan0\n\t\ttype AP\n\tInterface wlan1\n\t\ttype monitor\n",
			want: []string{"wlan0", "wlan1"},
		},
		{
			name:  "empty yields nothing",
			input: "",
			want:  nil,
		},
		{
			name:  "no Interface lines yields nothing",
			input: "phy#0\n\tno interfaces here\n",
			want:  nil,
		},
		{
			name:  "hostile name preserved verbatim",
			input: "phy#0\n\tInterface my wifi;$(evil)\n\t\ttype managed\n",
			want:  []string{"my wifi;$(evil)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseWiFiInterfaces(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseWiFiInterfaces = %v, want %v", got, tt.want)
			}
		})
	}
}
