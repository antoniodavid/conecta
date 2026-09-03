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

// Status returns the current VPN status
func (m *Manager) Status() (*Status, error) {
	s := &Status{Interface: m.iface}

	// Check interface
	out, err := exec.Command("ip", "-4", "addr", "show", m.iface).Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "inet ") {
				parts := strings.Fields(line)
				for i, p := range parts {
					if p == "inet" && i+1 < len(parts) {
						s.IP = parts[i+1]
						s.Connected = true
						break
					}
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
