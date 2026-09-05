package vpn

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager handles WireGuard VPN operations
type Manager struct {
	iface string
	name  string
}

// NewManager creates a new VPN manager
func NewManager(iface, name string) *Manager {
	if iface == "" {
		iface = "wg0"
	}
	if name == "" {
		name = "USA"
	}
	return &Manager{iface: iface, name: name}
}

// parseWireguardLinks parses `ip -o link show type wireguard` output and
// returns the names of UP interfaces in order.
func parseWireguardLinks(output string) []string {
	var ifaces []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Format: "<idx>: <name>: <flags> ..." e.g.
		// "633: USA: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 ..."
		parts := strings.SplitN(trimmed, ":", 3)
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		flags := parts[2]
		start := strings.Index(flags, "<")
		end := strings.Index(flags, ">")
		if start < 0 || end < 0 || end <= start {
			continue
		}
		up := false
		for _, f := range strings.Split(flags[start+1:end], ",") {
			if strings.TrimSpace(f) == "UP" {
				up = true
				break
			}
		}
		if !up {
			continue
		}
		ifaces = append(ifaces, name)
	}
	return ifaces
}

// firstIPv4 returns the first "inet <addr>" value in `ip addr` output
// (CIDR form, e.g. "10.14.0.2/16"), or "" when there is none.
func firstIPv4(output string) string {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "inet" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}
	return ""
}

// Status returns the current VPN status
func (m *Manager) Status() (*Status, error) {
	s := &Status{Interface: m.iface}

	// Check the configured interface first.
	if out, err := exec.Command("ip", "-4", "addr", "show", m.iface).Output(); err == nil {
		if ip := firstIPv4(string(out)); ip != "" {
			s.IP = ip
			s.Connected = true
		}
	}

	// Fall back: adopt the first UP wireguard interface carrying IPv4, so a
	// renamed iface (e.g. USA instead of wg0) still reports connected.
	if !s.Connected {
		if out, err := exec.Command("ip", "-o", "link", "show", "type", "wireguard").Output(); err == nil {
			for _, name := range parseWireguardLinks(string(out)) {
				addrOut, err := exec.Command("ip", "-4", "-o", "addr", "show", "dev", name).Output()
				if err != nil {
					continue
				}
				if ip := firstIPv4(string(addrOut)); ip != "" {
					s.Interface = name
					s.IP = ip
					s.Connected = true
					break
				}
			}
		}
	}

	// Get connection name from NetworkManager
	if s.Connected {
		out, _ := exec.Command("nmcli", "-t", "-f", "NAME,TYPE", "con", "show", "--active").Output()
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "wireguard") {
				parts := strings.Split(line, ":")
				if len(parts) > 0 {
					s.ConnectionName = parts[0]
				}
				break
			}
		}
	}

	return s, nil
}

// Connect activates the VPN
func (m *Manager) Connect() error {
	// Try NetworkManager first
	cmd := exec.Command("nmcli", "con", "up", m.name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Try wg-quick as fallback
		cmd = exec.Command("sudo", "wg-quick", "up", m.iface)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to connect VPN: %s", string(out))
		}
	}
	return nil
}

// Disconnect deactivates the VPN
func (m *Manager) Disconnect() error {
	// Try NetworkManager first
	cmd := exec.Command("nmcli", "con", "down", m.name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Try wg-quick as fallback
		cmd = exec.Command("sudo", "wg-quick", "down", m.iface)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to disconnect VPN: %s", string(out))
		}
	}
	return nil
}

// Toggle switches VPN on/off
func (m *Manager) Toggle() (bool, error) {
	status, err := m.Status()
	if err != nil {
		return false, err
	}

	if status.Connected {
		err = m.Disconnect()
		return false, err
	}

	err = m.Connect()
	return true, err
}

// IsRunning checks if the VPN interface is up
func (m *Manager) IsRunning() bool {
	out, err := exec.Command("ip", "link", "show", m.iface).Output()
	return err == nil && strings.Contains(string(out), m.iface)
}
