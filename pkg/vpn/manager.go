package vpn

import (
	"fmt"
	"os"
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

// unescapeNM restores nmcli -t escaping (\: -> :, \\ -> \).
func unescapeNM(s string) string {
	s = strings.ReplaceAll(s, `\:`, ":")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// parseNMConList parses `nmcli -t -f NAME,UUID,TYPE con show` output and
// returns wireguard profile names in order. Real shape incl:
// "USA:0544e4d0-3bf8-4ae4-873c-14fedd922c74:wireguard".
func parseNMConList(output string) []string {
	out := []string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.Split(trimmed, ":")
		if len(parts) < 3 {
			continue
		}
		if parts[len(parts)-1] != "wireguard" {
			continue
		}
		name := unescapeNM(strings.Join(parts[:len(parts)-2], ":"))
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

// parseNMActiveDevices parses `nmcli -t -f NAME,DEVICE con show --active`
// into name -> device. Device "--" or empty maps to "".
func parseNMActiveDevices(output string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.Split(trimmed, ":")
		if len(parts) < 2 {
			continue
		}
		dev := parts[len(parts)-1]
		name := unescapeNM(strings.Join(parts[:len(parts)-1], ":"))
		if name == "" {
			continue
		}
		if dev == "--" {
			dev = ""
		}
		m[name] = dev
	}
	return m
}

// parseImportedName extracts the connection name from
// `nmcli connection import` output, e.g.
// "Connection 'USA' (uuid ...) successfully added." -> "USA".
func parseImportedName(output string) string {
	for _, q := range []struct{ open, close string }{{"'", "'"}, {`"`, `"`}} {
		if i := strings.Index(output, q.open); i >= 0 {
			rest := output[i+len(q.open):]
			if j := strings.Index(rest, q.close); j > 0 {
				return rest[:j]
			}
		}
	}
	return ""
}

// validateImportFile checks the path exists, is a file, and looks like a
// WireGuard conf (contains an [Interface] section header line).
func validateImportFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot read file %q: %v", path, err)
	}
	if fi.IsDir() {
		return fmt.Errorf("not a WireGuard conf %q: is a directory", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read file %q: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "[Interface]" {
			return nil
		}
	}
	return fmt.Errorf("not a WireGuard conf %q: missing [Interface] section", path)
}

// ListProfiles returns all NetworkManager wireguard profiles with active
// state, device, and IPv4 (when active). Empty store yields an empty slice,
// never an error for zero profiles.
func (m *Manager) ListProfiles() ([]Profile, error) {
	out, err := exec.Command("nmcli", "-t", "-f", "NAME,UUID,TYPE", "con", "show").Output()
	if err != nil {
		return nil, fmt.Errorf("tool unavailable: nmcli con show: %v", err)
	}
	names := parseNMConList(string(out))
	activeOut, err := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE", "con", "show", "--active").Output()
	active := map[string]string{}
	if err == nil {
		active = parseNMActiveDevices(string(activeOut))
	}
	profiles := make([]Profile, 0, len(names))
	for _, name := range names {
		p := Profile{Name: name, Country: CountryCode(name), Flag: CountryFlag(name)}
		if dev, ok := active[name]; ok {
			p.Active = true
			p.Device = dev
			if dev == "" {
				dev = name
			}
			if addrOut, err := exec.Command("ip", "-4", "-o", "addr", "show", "dev", dev).Output(); err == nil {
				p.IP = firstIPv4(string(addrOut))
			}
			if p.Device == "" {
				p.Device = name
			}
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
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

	// Best-effort profile list for `vpn status` (never fails status).
	if profiles, err := m.ListProfiles(); err == nil {
		s.Profiles = profiles
	} else {
		s.Profiles = make([]Profile, 0)
	}

	return s, nil
}

// Connect activates the VPN using the configured profile name.
func (m *Manager) Connect() error {
	return m.ConnectTo(m.name)
}

// ConnectTo activates the named NetworkManager profile. The wg-quick
// fallback only applies to the configured profile (it needs the local
// interface name, not an arbitrary NM profile name).
func (m *Manager) ConnectTo(name string) error {
	cmd := exec.Command("nmcli", "con", "up", name)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if name != m.name {
		return fmt.Errorf("failed to connect VPN: %s", string(out))
	}
	cmd = exec.Command("sudo", "wg-quick", "up", m.iface)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect VPN: %s", string(out))
	}
	return nil
}

// Import validates a WireGuard conf file and imports it via NetworkManager.
// It returns the added connection name parsed from nmcli output (falling
// back to the file base name without .conf when nmcli prints no quoted
// name). It never deletes or overwrites: duplicate names are refused by
// nmcli itself and its message is surfaced. Any failure is an error.
func (m *Manager) Import(path string) (string, error) {
	if err := validateImportFile(path); err != nil {
		return "", err
	}
	out, err := exec.Command("nmcli", "connection", "import", "type", "wireguard", "file", path).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("nmcli import failed: %s: %v", strings.TrimSpace(string(out)), err)
	}
	if name := parseImportedName(string(out)); name != "" {
		return name, nil
	}
	base := path
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".conf"), nil
}

// Disconnect deactivates the VPN without deleting anything:
// `nmcli con down` only deactivates the profile (it stays listed by
// `vpn list` as inactive), and the `wg-quick down` fallback only removes
// the runtime interface, never the conf file.
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
