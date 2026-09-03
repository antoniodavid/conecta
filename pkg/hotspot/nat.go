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

// Setup configures NAT rules
func (n *NAT) Setup() error {
	if err := checkAuthz(); err != nil {
		return err
	}
	// Enable IP forwarding
	if err := exec.Command("sudo", "sysctl", "-w", "net.ipv4.ip_forward=1").Run(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Determine exit interface
	exitIface := n.exitIface
	if n.vpnIface != "" && interfaceExists(n.vpnIface) {
		exitIface = n.vpnIface
	}

	// Flush existing rules first
	n.flushRules(exitIface)

	// Add NAT rules
	rules := [][]string{
		{"-t", "nat", "-A", "POSTROUTING", "-s", n.hotspotSubnet, "-o", exitIface, "-j", "MASQUERADE"},
		{"-A", "FORWARD", "-i", n.hotspotIface, "-o", exitIface, "-j", "ACCEPT"},
		{"-A", "FORWARD", "-i", exitIface, "-o", n.hotspotIface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	}

	for _, args := range rules {
		cmd := exec.Command("sudo", append([]string{"iptables"}, args...)...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add iptables rule: %w", err)
		}
	}

	return nil
}

// Cleanup removes NAT rules
func (n *NAT) Cleanup() {
	exitIface := n.exitIface
	if n.vpnIface != "" && interfaceExists(n.vpnIface) {
		exitIface = n.vpnIface
	}

	n.flushRules(exitIface)
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
	out, err = exec.Command("sudo", "iptables", "-t", "nat", "-L", "POSTROUTING", "-n").Output()
	if err == nil {
		s.NATRules = strings.Count(string(out), "MASQUERADE")
	}

	// Check FORWARD rules
	out, err = exec.Command("sudo", "iptables", "-L", "FORWARD", "-n").Output()
	if err == nil {
		s.ForwardRules = strings.Count(string(out), "ACCEPT")
	}

	return s, nil
}

func (n *NAT) flushRules(exitIface string) {
	rules := [][]string{
		{"-t", "nat", "-D", "POSTROUTING", "-s", n.hotspotSubnet, "-o", exitIface, "-j", "MASQUERADE"},
		{"-D", "FORWARD", "-i", n.hotspotIface, "-o", exitIface, "-j", "ACCEPT"},
		{"-D", "FORWARD", "-i", exitIface, "-o", n.hotspotIface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	}

	for _, args := range rules {
		exec.Command("sudo", append([]string{"iptables"}, args...)...).Run()
	}
}

// NATStatus represents the current NAT configuration
type NATStatus struct {
	IPForward   bool
	NATRules    int
	ForwardRules int
}

// interfaceExists checks if a network interface exists
func interfaceExists(name string) bool {
	out, err := exec.Command("ip", "link", "show", name).Output()
	return err == nil && strings.Contains(string(out), name)
}
