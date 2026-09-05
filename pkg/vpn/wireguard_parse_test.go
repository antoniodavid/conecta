package vpn

import (
	"reflect"
	"testing"
)

// Table-driven wireguard link parse incl. real host output (USA iface).
func TestParseWireguardLinks(t *testing.T) {
	realLink := "633: USA: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000\\    link/none \n"
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "real ip -o link output yields USA",
			input: realLink,
			want:  []string{"USA"},
		},
		{
			name:  "DOWN wireguard iface skipped",
			input: "2: wg0: <POINTOPOINT,MULTICAST,NOARP> mtu 1420 qdisc noop state DOWN mode DEFAULT group default qlen 1000\\    link/none\n",
			want:  nil,
		},
		{
			name: "multiple links keep UP order",
			input: "2: wg0: <POINTOPOINT,MULTICAST,NOARP> mtu 1420 qdisc noop state DOWN mode DEFAULT group default qlen 1000\\    link/none\n" +
				"633: USA: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000\\    link/none \n" +
				"634: wg1: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000\\    link/none \n",
			want: []string{"USA", "wg1"},
		},
		{
			name:  "empty yields nothing",
			input: "",
			want:  nil,
		},
		{
			name:  "non-wireguard garbage yields nothing",
			input: "not a link line\n",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWireguardLinks(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseWireguardLinks = %v, want %v", got, tt.want)
			}
		})
	}
}

// Table-driven IPv4 pick incl. real host outputs.
func TestFirstIPv4(t *testing.T) {
	realOneLine := "633: USA    inet 10.14.0.2/16 brd 10.14.255.255 scope global noprefixroute USA\\       valid_lft forever preferred_lft forever\n"
	realMultiLine := "633: USA: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN group default qlen 1000\n    inet 10.14.0.2/16 brd 10.14.255.255 scope global noprefixroute USA\n       valid_lft forever preferred_lft forever\n"
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "real ip -4 -o addr yields CIDR",
			input: realOneLine,
			want:  "10.14.0.2/16",
		},
		{
			name:  "real ip -4 addr multiline yields CIDR",
			input: realMultiLine,
			want:  "10.14.0.2/16",
		},
		{
			name:  "picks first inet",
			input: "2: wg0    inet 10.0.0.1/24 scope global wg0\n2: wg0    inet 10.0.0.2/24 scope global secondary wg0\n",
			want:  "10.0.0.1/24",
		},
		{
			name:  "no inet yields empty",
			input: "633: USA: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN group default qlen 1000\n",
			want:  "",
		},
		{
			name:  "empty yields empty",
			input: "",
			want:  "",
		},
		{
			name:  "missing device yields empty",
			input: "Device \"wg0\" does not exist.\n",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstIPv4(tt.input); got != tt.want {
				t.Fatalf("firstIPv4 = %q, want %q", got, tt.want)
			}
		})
	}
}
