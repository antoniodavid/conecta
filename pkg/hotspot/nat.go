package hotspot

import (
	"fmt"
	"os/exec"
	"strings"
)

// NAT manages iptables NAT rules for hotspot
type NAT struct {
	hotspotSubnet string
	hotspotIface  string
	vpnIface      string
	exitIface     string
}

// NewNAT creates a new NAT manager
func NewNAT(hotspotSubnet, hotspotIface, vpnIface, exitIface string) *NAT {
	if hotspotSubnet == "" {
		hotspotSubnet = "192.168.12.0/24"
	}
	if hotspotIface == "" {
		hotspotIface = "ap0"
	}
	if exitIface == "" {
		exitIface = "enp3s0"
	}

	return &NAT{
		hotspotSubnet: hotspotSubnet,
		hotspotIface:  hotspotIface,
		vpnIface:      vpnIface,
		exitIface:     exitIface,
	}
}

// exitIfaceFor resolves the exit interface: prefer the VPN interface when it
// exists (hotspot clients exit through the tunnel), otherwise the physical
// uplink. Rules are written and removed against the same resolved value.
func (n *NAT) exitIfaceFor() string {
	if n.vpnIface != "" && interfaceExists(n.vpnIface) {
		return n.vpnIface
	}
	return n.exitIface
}

// Setup configures NAT rules
func (n *NAT) Setup() error {
	if err := checkAuthz(); err != nil {
		return err
	}
	// Enable IP forwarding
	if err := exec.Command("sudo", "-n", "sysctl", "-w", "net.ipv4.ip_forward=1").Run(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	exitIface := n.exitIfaceFor()

	// Drop stale rules first, then add the current set. A stale rule with a
	// different exit iface is absent, not an error; a denied/flaky sudo IS an
	// error and must fail before we add anything new.
	if err := n.flushRules(exitIface); err != nil {
		return err
	}

	rules := [][]string{
		{"-t", "nat", "-A", "POSTROUTING", "-s", n.hotspotSubnet, "-o", exitIface, "-j", "MASQUERADE"},
		{"-A", "FORWARD", "-i", n.hotspotIface, "-o", exitIface, "-j", "ACCEPT"},
		{"-A", "FORWARD", "-i", exitIface, "-o", n.hotspotIface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	}

	for _, args := range rules {
		cmd := exec.Command("sudo", append([]string{"-n", "iptables"}, args...)...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add iptables rule: %w", err)
		}
	}

	return nil
}

// Cleanup removes NAT rules. It fails closed: a rule already absent is not an
// error, but a real failure (denied sudo, iptables unavailable) is surfaced.
func (n *NAT) Cleanup() error {
	return n.flushRules(n.exitIfaceFor())
}

// Status returns current NAT status
func (n *NAT) Status() (*NATStatus, error) {
	s := &NATStatus{}

	// Check IP forwarding
	out, err := exec.Command("cat", "/proc/sys/net/ipv4/ip_forward").Output()
	if err == nil {
		s.IPForward = strings.TrimSpace(string(out)) == "1"
	}

	// Count NAT rules
	out, err = exec.Command("sudo", "-n", "iptables", "-t", "nat", "-L", "POSTROUTING", "-n").Output()
	if err == nil {
		s.NATRules = strings.Count(string(out), "MASQUERADE")
	}

	// Check FORWARD rules
	out, err = exec.Command("sudo", "-n", "iptables", "-L", "FORWARD", "-n").Output()
	if err == nil {
		s.ForwardRules = strings.Count(string(out), "ACCEPT")
	}

	return s, nil
}

// ruleAbsent reports whether an iptables -D failure means "the rule was not
// there" (which is not an error) rather than a real failure (denied sudo or a
// missing tool). The exact wording varies across iptables versions.
func ruleAbsent(msg string) bool {
	return strings.Contains(msg, "Bad rule") ||
		strings.Contains(msg, "does a matching rule exist") ||
		strings.Contains(msg, "No such file or directory")
}

// flushRules removes the NAT rules for the given exit interface. A rule that
// is already absent is ignored; any other failure (denied sudo, missing
// iptables) is collected and returned.
func (n *NAT) flushRules(exitIface string) error {
	rules := [][]string{
		{"-t", "nat", "-D", "POSTROUTING", "-s", n.hotspotSubnet, "-o", exitIface, "-j", "MASQUERADE"},
		{"-D", "FORWARD", "-i", n.hotspotIface, "-o", exitIface, "-j", "ACCEPT"},
		{"-D", "FORWARD", "-i", exitIface, "-o", n.hotspotIface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	}

	var failures []string
	for _, args := range rules {
		cmd := exec.Command("sudo", append([]string{"-n", "iptables"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil && !ruleAbsent(string(out)) {
			failures = append(failures, fmt.Sprintf("%s: %s", strings.TrimSpace(string(out)), err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("failed to remove NAT rules: %s", strings.Join(failures, "; "))
	}
	return nil
}

// NATStatus represents the current NAT configuration
type NATStatus struct {
	IPForward    bool
	NATRules     int
	ForwardRules int
}

// interfaceExists checks if a network interface exists
func interfaceExists(name string) bool {
	out, err := exec.Command("ip", "link", "show", name).Output()
	return err == nil && strings.Contains(string(out), name)
}
