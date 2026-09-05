package vpn

import (
	"reflect"
	"testing"
)

func TestParseNMConList(t *testing.T) {
	// Real host shape: NAME:UUID:TYPE with two wireguard profiles.
	real := "EXTR@ 2:b1eaeb4e-d6f3-41ab-bc33-92826732778d:802-11-wireless\n" +
		"USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard\n" +
		"Binhex-Spain:f2ef384a-3fc2-431b-baef-bb1245a76c97:wireguard\n" +
		"Wired connection 1:e843bf54-83c1-3612-ae7e-5b999df6f95f:802-3-ethernet\n"
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"real output yields wireguard names in order", real, []string{"USA", "Binhex-Spain"}},
		{"no wireguard yields empty", "lo:63e55428-db82-4588-b9de-3d78f5c6a130:loopback\n", []string{}},
		{"empty yields empty", "", []string{}},
		{"garbage skipped", "not a profile\n", []string{}},
		{"colon in name rejoined", `a\:b:11111111-1111-1111-1111-111111111111:wireguard` + "\n", []string{"a:b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNMConList(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseNMConList = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNMActiveDevices(t *testing.T) {
	// Real host shape: NAME:DEVICE for active connections.
	real := "EXTR@ 2:wlo1\nUSA:USA\nlo:lo\ndocker0:docker0\n"
	got := parseNMActiveDevices(real)
	if got["USA"] != "USA" {
		t.Fatalf("active USA device = %q, want USA", got["USA"])
	}
	if got["EXTR@ 2"] != "wlo1" {
		t.Fatalf("active wifi device = %q, want wlo1", got["EXTR@ 2"])
	}
	if got := parseNMActiveDevices("USA:--\n"); got["USA"] != "" {
		t.Fatalf("device -- must map to empty, got %q", got["USA"])
	}
	if got := parseNMActiveDevices(""); len(got) != 0 {
		t.Fatalf("empty must yield empty map, got %v", got)
	}
}
