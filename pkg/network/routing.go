package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// Routing manages network routing and NAT
type Routing struct {
	config *NetworkConfig
}

// NewRouting creates a new routing manager
func NewRouting(config *NetworkConfig) *Routing {
	if config == nil {
		config = DefaultConfig()
	}
	return &Routing{config: config}
}

// EnsureRoute adds the route to ETECSA network if not present
func (r *Routing) EnsureRoute() error {
	// Check if route exists
	out, err := exec.Command("ip", "route", "show", "10.180.0.0/16").Output()
	if err == nil && strings.Contains(string(out), "10.180.0.0/16") {
		return nil // Route already exists
	}

	// Add route
	cmd := exec.Command("sudo", "-n", "ip", "route", "add",
		"10.180.0.0/16", "via", r.config.Gateway, "dev", r.config.Interface)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	return nil
}

// RemoveRoute removes the ETECSA route
func (r *Routing) RemoveRoute() error {
	cmd := exec.Command("sudo", "-n", "ip", "route", "del",
		"10.180.0.0/16", "via", r.config.Gateway, "dev", r.config.Interface)
	if err := cmd.Run(); err != nil {
		// Route might not exist, that's OK
		return nil
	}
	return nil
}

// CheckGateway checks if the gateway is reachable
func (r *Routing) CheckGateway() bool {
	out, err := exec.Command("ping", "-c", "1", "-W", "2", r.config.Gateway).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "1 received")
}

// SetupNAT configures NAT for hotspot clients
func (r *Routing) SetupNAT(hotspotSubnet, hotspotIface, vpnIface string) error {
	// Enable IP forwarding
	if err := exec.Command("sudo", "sysctl", "-w", "net.ipv4.ip_forward=1").Run(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Flush existing rules
	r.flushNATRules(hotspotSubnet, hotspotIface, vpnIface)

	// Determine exit interface
	exitIface := r.config.Interface
	if vpnIface != "" && interfaceExists(vpnIface) {
		exitIface = vpnIface
	}

	// Add NAT rules
	rules := [][]string{
		{"-t", "nat", "-A", "POSTROUTING", "-s", hotspotSubnet, "-o", exitIface, "-j", "MASQUERADE"},
		{"-A", "FORWARD", "-i", hotspotIface, "-o", exitIface, "-j", "ACCEPT"},
		{"-A", "FORWARD", "-i", exitIface, "-o", hotspotIface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	}

	for _, args := range rules {
		cmd := exec.Command("sudo", append([]string{"iptables"}, args...)...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add iptables rule: %w", err)
		}
	}

	return nil
}

// CleanupNAT removes NAT rules
func (r *Routing) CleanupNAT(hotspotSubnet, hotspotIface, vpnIface string) {
	r.flushNATRules(hotspotSubnet, hotspotIface, vpnIface)
}

func (r *Routing) flushNATRules(hotspotSubnet, hotspotIface, vpnIface string) {
	exitIface := r.config.Interface
	if vpnIface != "" && interfaceExists(vpnIface) {
		exitIface = vpnIface
	}

	rules := [][]string{
		{"-t", "nat", "-D", "POSTROUTING", "-s", hotspotSubnet, "-o", exitIface, "-j", "MASQUERADE"},
		{"-D", "FORWARD", "-i", hotspotIface, "-o", exitIface, "-j", "ACCEPT"},
		{"-D", "FORWARD", "-i", exitIface, "-o", hotspotIface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	}

	for _, args := range rules {
		exec.Command("sudo", append([]string{"iptables"}, args...)...).Run()
	}
}

// interfaceExists checks if a network interface exists
func interfaceExists(name string) bool {
	out, err := exec.Command("ip", "link", "show", name).Output()
	return err == nil && strings.Contains(string(out), name)
}
