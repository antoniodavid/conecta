package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// PortalNet is the ETECSA captive-portal network. It is a private net
// reachable through the physical gateway, so a full-tunnel VPN would
// blackhole it without an explicit route.
const PortalNet = "10.180.0.0/16"

// EnsurePortalRoute makes sure PortalNet stays routed via the physical
// gateway while the VPN is up, mirroring etecsa-tui
// (`ip route add 10.180.0.0/16 via <gateway> dev <iface>`). The check runs
// `ip route show` first (read-only, no sudo) and returns nil when the route
// is already present, so the function is idempotent. Otherwise the route is
// added with one fixed-argv sudo command (never a shell); on a failed add
// the route is re-checked once to absorb the race where it appears between
// the check and the add. Only a still-absent route is an error, and the
// error carries the sudo output. gateway/iface are required.
func EnsurePortalRoute(gateway, iface string) error {
	if gateway == "" || iface == "" {
		return fmt.Errorf("portal route: gateway and interface must not be empty")
	}
	present := func() bool {
		out, err := exec.Command("ip", "route", "show", PortalNet).Output()
		return err == nil && strings.Contains(string(out), PortalNet)
	}
	if present() {
		return nil // already routed; nothing to do
	}
	cmd := exec.Command("sudo", "-n", "ip", "route", "add", PortalNet, "via", gateway, "dev", iface)
	if out, err := cmd.CombinedOutput(); err != nil {
		if present() {
			return nil // lost the race: the route is up despite the failed add
		}
		return fmt.Errorf("failed to add portal route %s via %s dev %s: %s: %v",
			PortalNet, gateway, iface, strings.TrimSpace(string(out)), err)
	}
	return nil
}
