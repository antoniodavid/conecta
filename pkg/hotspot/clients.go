package hotspot

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrInvalidMAC reports a MAC that is not aa:bb:cc:dd:ee:ff hex.
var ErrInvalidMAC = errors.New("invalid MAC address (expected aa:bb:cc:dd:ee:ff)")

// Client represents a connected hotspot client
type Client struct {
	IP      string
	MAC     string
	Name    string
	RXBytes uint64
	TXBytes uint64
}

// ClientManager handles client detection and monitoring
type ClientManager struct {
	iface string
}

// NewClientManager creates a new client manager
func NewClientManager(iface string) *ClientManager {
	if iface == "" {
		iface = "ap0"
	}
	return &ClientManager{iface: iface}
}

// ListClients returns all connected clients
func (cm *ClientManager) ListClients() ([]Client, error) {
	// Method 1: iw station dump (most reliable). Some drivers (including
	// this host's AP iface) return an empty dump even with clients attached,
	// so an empty iw result is never treated as authoritative.
	clients, err := cm.listFromIw()

	// Method 2: DHCP leases. Read always; they are the fallback source when
	// iw reports nothing (empty dump or error).
	leases := cm.readDHCPLeases()
	if err != nil || len(clients) == 0 {
		clients = leasesToClients(leases)
	}

	// Enrich with DHCP leases
	for i := range clients {
		for _, l := range leases {
			if strings.EqualFold(clients[i].MAC, l.mac) {
				clients[i].IP = l.ip
				clients[i].Name = l.name
				break
			}
		}
	}

	// Enrich with ARP table if IP still missing (runs even when iw returned
	// nothing, so lease-only listings still reach the ARP table).
	cm.enrichFromARP(clients)

	return clients, nil
}

// CountClients returns the number of connected clients
func (cm *ClientManager) CountClients() int {
	out, err := exec.Command("iw", "dev", cm.iface, "station", "dump").Output()
	if err != nil {
		return 0
	}
	return strings.Count(string(out), "Station ")
}

func (cm *ClientManager) listFromIw() ([]Client, error) {
	var clients []Client

	out, err := exec.Command("iw", "dev", cm.iface, "station", "dump").Output()
	if err != nil {
		return nil, err
	}

	stations := strings.Split(string(out), "Station ")
	for _, block := range stations[1:] { // skip first empty split
		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		mac := strings.TrimSpace(strings.Fields(lines[0])[0])
		c := Client{MAC: strings.ToUpper(mac)}

		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "RX bytes:") {
				if v, err := parseIwBytes(line); err == nil {
					c.RXBytes = v
				}
			} else if strings.HasPrefix(line, "TX bytes:") {
				if v, err := parseIwBytes(line); err == nil {
					c.TXBytes = v
				}
			}
		}

		clients = append(clients, c)
	}

	return clients, nil
}

type dhcpLease struct {
	ip   string
	mac  string
	name string
}

// parseLeaseLine parses one dnsmasq lease line. Stock dnsmasq writes
// "expiry ip mac name"; create_ap on this host writes "expiry mac ip name"
// (verified against the live lease file). Detect the fields by shape so
// both orders parse correctly.
func parseLeaseLine(line string) (dhcpLease, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return dhcpLease{}, false
	}
	var ip, mac string
	for _, f := range fields[1:3] {
		switch {
		case validMAC(f):
			mac = strings.ToUpper(f)
		case isIPv4Field(f):
			ip = f
		}
	}
	if ip == "" || mac == "" {
		return dhcpLease{}, false
	}
	name := fields[3]
	if name == "*" {
		name = ""
	}
	return dhcpLease{ip: ip, mac: mac, name: name}, true
}

// isIPv4Field reports whether s looks like an IPv4 address.
func isIPv4Field(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// leasesToClients maps DHCP leases to the public Client shape.
func leasesToClients(leases []dhcpLease) []Client {
	clients := make([]Client, 0, len(leases))
	for _, l := range leases {
		clients = append(clients, Client{MAC: l.mac, IP: l.ip, Name: l.name})
	}
	return clients
}

func (cm *ClientManager) readDHCPLeases() []dhcpLease {
	var leases []dhcpLease

	// Find latest create_ap temp dir. The iface in the dir name varies
	// (wlan0, wlo1, ...), so match any wireless iface.
	tmpDirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "create_ap.*.conf.*"))
	var latestDir string
	var latestTime int64

	for _, d := range tmpDirs {
		fi, err := os.Stat(d)
		if err == nil && fi.ModTime().Unix() > latestTime {
			latestTime = fi.ModTime().Unix()
			latestDir = d
		}
	}

	if latestDir == "" {
		return leases
	}

	data, err := os.ReadFile(filepath.Join(latestDir, "dnsmasq.leases"))
	if err != nil {
		return leases
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if l, ok := parseLeaseLine(scanner.Text()); ok {
			leases = append(leases, l)
		}
	}

	return leases
}

func (cm *ClientManager) enrichFromARP(clients []Client) {
	out, err := exec.Command("ip", "neigh", "show", "dev", cm.iface).Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// "192.168.12.129 dev ap0 lladdr 22:df:51:de:e8:40 REACHABLE"
		if len(fields) >= 5 {
			mac := strings.ToUpper(fields[4])
			for i := range clients {
				if clients[i].MAC == mac && clients[i].IP == "" {
					clients[i].IP = fields[0]
				}
			}
		}
	}
}

// KickClient deauthenticates a hotspot client via `iw station del`. This
// requires root, so it runs through sudo with a fixed argv covered by the
// conecta sudoers drop-in (contrib/sudoers-conecta). An empty iface is
// resolved to the live AP interface, falling back to "ap0". No shell.
func KickClient(iface, mac string) error {
	if !validMAC(mac) {
		return ErrInvalidMAC
	}
	if iface == "" {
		if iface = apInterface(); iface == "" {
			iface = "ap0"
		}
	}
	out, err := exec.Command("sudo", "iw", "dev", iface, "station", "del", mac, "subtype", "0xC").CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("kick %s from %s failed: %w (%s)", mac, iface, err, msg)
		}
		return fmt.Errorf("kick %s from %s failed: %w", mac, iface, err)
	}
	return nil
}

// validMAC reports whether mac looks like aa:bb:cc:dd:ee:ff (hex).
func validMAC(mac string) bool {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return false
	}
	for _, p := range parts {
		if len(p) != 2 {
			return false
		}
		for _, c := range p {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

func parseIwBytes(line string) (uint64, error) {
	// "RX bytes: 1234567" -> 1234567
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		return strconv.ParseUint(parts[2], 10, 64)
	}
	return 0, strconv.ErrSyntax
}
