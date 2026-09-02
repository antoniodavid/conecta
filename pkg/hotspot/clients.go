package hotspot

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

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
	var clients []Client

	// Method 1: iw station dump (most reliable)
	iwClients, err := cm.listFromIw()
	if err == nil {
		clients = append(clients, iwClients...)
	}

	// Enrich with DHCP leases
	leases := cm.readDHCPLeases()
	for i := range clients {
		for _, l := range leases {
			if strings.EqualFold(clients[i].MAC, l.mac) {
				clients[i].IP = l.ip
				clients[i].Name = l.name
				break
			}
		}
	}

	// Enrich with ARP table if IP still missing
	if len(clients) > 0 {
		cm.enrichFromARP(clients)
	}

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

func (cm *ClientManager) readDHCPLeases() []dhcpLease {
	var leases []dhcpLease

	// Find latest create_ap temp dir
	tmpDirs, _ := filepath.Glob("/tmp/create_ap.wlan*.conf.*")
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
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			name := fields[3]
			if name == "*" {
				name = ""
			}
			leases = append(leases, dhcpLease{
				ip:   fields[1],
				mac:  strings.ToUpper(fields[2]),
				name: name,
			})
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
		if len(fields) >= 5 {
			mac := strings.ToUpper(fields[3])
			for i := range clients {
				if clients[i].MAC == mac && clients[i].IP == "" {
					clients[i].IP = fields[0]
				}
			}
		}
	}
}

func parseIwBytes(line string) (uint64, error) {
	// "RX bytes: 1234567" -> 1234567
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		return strconv.ParseUint(parts[2], 10, 64)
	}
	return 0, strconv.ErrSyntax
}
