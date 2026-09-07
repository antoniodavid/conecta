package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"conecta/pkg/config"
	"conecta/pkg/hotspot"
	"conecta/pkg/network"
	"conecta/pkg/vpn"
)

func main() {
	if len(os.Args) < 2 {
		emitError("invalid_input", "usage: conecta-cli <status|login|logout|speed|hotspot|nat|vpn|help>")
		return
	}
	command := os.Args[1]
	rest := os.Args[2:]

	// Load config first; malformed config fails without partial action.
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		emitError("config", fmt.Sprintf("cannot load config: %v", err))
		return
	}
	if err := validateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		emitError("config", err.Error())
		return
	}

	switch command {
	case "help", "--help", "-h":
		printUsage()
	case "status":
		rejectSubcommandFlags(command, rest)
		cmdStatus(cfg)
	case "login":
		cmdLogin(cfg, rest)
	case "logout":
		rejectSubcommandFlags(command, rest)
		cmdLogout(cfg)
	case "speed":
		rejectSubcommandFlags(command, rest)
		cmdSpeed(cfg)
	case "hotspot":
		cmdHotspot(cfg, rest)
	case "nat":
		cmdNAT(cfg, rest)
	case "vpn":
		cmdVPN(cfg, rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		emitError("invalid_input", fmt.Sprintf("unknown command: %s", command))
	}
}

func printUsage() {
	fmt.Println(helpText)
}

// hasLoginFlags reports whether args contain the login-only --user/--pass flags.
func hasLoginFlags(args []string) bool {
	for _, a := range args {
		if a == "--user" || strings.HasPrefix(a, "--user=") ||
			a == "--pass" || strings.HasPrefix(a, "--pass=") {
			return true
		}
	}
	return false
}

// rejectSubcommandFlags ensures --user/--pass only bind to login.
func rejectSubcommandFlags(cmd string, args []string) {
	if hasLoginFlags(args) {
		fmt.Fprintf(os.Stderr, "--user/--pass only apply to login\n")
		emitError("invalid_input", "--user/--pass only apply to login")
	}
}

// validateConfig fails without panic or partial action on bad URLs/values.
func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	u, err := url.Parse(cfg.Network.PortalURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid portal_url: %q", cfg.Network.PortalURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid portal_url scheme: %q", u.Scheme)
	}
	if cfg.Network.Gateway == "" || cfg.Network.Interface == "" {
		return fmt.Errorf("network gateway/interface must be set")
	}
	return nil
}

// emitOpError maps operational failures to the contract exits.
// Authz failures exit 4, unavailable tools exit 3 with code unavailable.
func emitOpError(op string, err error) {
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "authz:") || strings.Contains(lower, "denied") && strings.Contains(lower, "sudo"):
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", op, err)
		emitError("authz", msg)
	case strings.Contains(lower, "tool unavailable"):
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", op, err)
		emitError("unavailable", msg)
	default:
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", op, err)
		emitError("op_failed", msg)
	}
}

func cmdStatus(cfg *config.Config) {
	portal := network.NewPortal(&network.NetworkConfig{
		Gateway:   cfg.Network.Gateway,
		Interface: cfg.Network.Interface,
		PortalURL: cfg.Network.PortalURL,
	})

	conn, err := portal.CheckPortal()
	if err != nil {
		emitOpError("status", err)
		return
	}
	// CLI owns detection: signed Wi-Fi signal and configured VPN interface.
	// Detect the real wireless iface (e.g. wlo1); fall back to the configured
	// interface only when detection yields nothing.
	wifiIface := cfg.Network.Interface
	if detected, err := network.DetectWiFiInterface(); err == nil && detected != "" {
		wifiIface = detected
	}
	snap := network.SnapshotLink(wifiIface)
	vpnStatus, _ := vpn.NewManager(cfg.VPN.Interface, cfg.VPN.Name).Status()
	if vpnStatus == nil {
		vpnStatus = &vpn.Status{Interface: cfg.VPN.Interface}
	}
	profiles := vpnStatus.Profiles
	if profiles == nil {
		profiles = make([]vpn.Profile, 0)
	}
	emitResult(map[string]any{
		"status":    conn.Status.String(),
		"gateway":   conn.Gateway,
		"interface": conn.Interface,
		"username":  cfg.Credentials.Username,
		"wifi": map[string]any{
			"available":  snap.Available,
			"ssid":       snap.SSID,
			"signal_dbm": snap.SignalDBM,
			"signal":     snap.Signal,
		},
		"vpn": map[string]any{
			"connected": vpnStatus.Connected,
			"ip":        vpnStatus.IP,
			"name":      vpnStatus.ConnectionName,
			"interface": vpnStatus.Interface,
			"profiles":  profiles,
		},
	})
}

// parseLoginFlags binds --user/--pass to the login subcommand via its own FlagSet.
// Empty values mean "fall back to configured credentials"; rest holds positionals.
func parseLoginFlags(args []string) (user, pass string, rest []string, err error) {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	u := fs.String("user", "", "Username")
	p := fs.String("pass", "", "Password")
	// Flag errors go to stderr; contract failure goes to stdout with exit 2.
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return "", "", nil, err
	}
	return *u, *p, fs.Args(), nil
}

// vpnPortalHint explains the off-VPN-only portal to users who hit it with
// the VPN up. The ETECSA portal is a private IP reachable only off-VPN.
const vpnPortalHint = "hint: the portal is reachable only with the VPN down — run `conecta-cli vpn disconnect` first, then retry"

// vpnStatusCheck reports whether the VPN is connected. A variable (not a
// direct manager call) so tests inject a stub and never exec nmcli.
var vpnStatusCheck = func(iface, name string) bool {
	st, err := vpn.NewManager(iface, name).Status()
	return err == nil && st != nil && st.Connected
}

// withPortalVPNHint appends the disconnect-VPN guidance to portal-access
// failures when the VPN is up. Pure: vpnConnected is injected, so unit
// tests never touch nmcli. Non-portal errors pass through unchanged, and
// the envelope code/exit mapping is untouched (guidance lives in the
// message string only).
func withPortalVPNHint(err error, vpnConnected bool) error {
	if err == nil || !vpnConnected {
		return err
	}
	lower := strings.ToLower(err.Error())
	// GET-phase markers: both Login and Logout hit GET portal first, so a
	// VPN-up unreachable portal always surfaces through one of these.
	if strings.Contains(lower, "cannot access portal") ||
		strings.Contains(lower, "portal unreachable") {
		return fmt.Errorf("%s (%s)", err.Error(), vpnPortalHint)
	}
	return err
}

func cmdLogin(cfg *config.Config, args []string) {
	u, p, extra, err := parseLoginFlags(args)
	if err != nil {
		emitError("invalid_input", fmt.Sprintf("invalid login flags: %v", err))
		return
	}
	if len(extra) > 0 {
		emitError("invalid_input", fmt.Sprintf("unexpected login args: %v", extra))
		return
	}

	if u == "" {
		u = cfg.Credentials.Username
	}
	if p == "" {
		p = cfg.Credentials.Password
	}
	if u == "" || p == "" {
		fmt.Fprintf(os.Stderr, "login requires --user/--pass or configured credentials\n")
		emitError("invalid_input", "login requires --user/--pass or configured credentials")
		return
	}

	portal := network.NewPortal(&network.NetworkConfig{
		Gateway:   cfg.Network.Gateway,
		Interface: cfg.Network.Interface,
		PortalURL: cfg.Network.PortalURL,
	})

	conn, err := portal.Login(u, p)
	if err != nil {
		emitOpError("login", withPortalVPNHint(err, vpnStatusCheck(cfg.VPN.Interface, cfg.VPN.Name)))
		return
	}
	emitResult(map[string]any{"status": conn.Status.String()})
}

func cmdLogout(cfg *config.Config) {
	portal := network.NewPortal(&network.NetworkConfig{
		Gateway:   cfg.Network.Gateway,
		Interface: cfg.Network.Interface,
		PortalURL: cfg.Network.PortalURL,
	})

	if err := portal.Logout(); err != nil {
		emitOpError("logout", withPortalVPNHint(err, vpnStatusCheck(cfg.VPN.Interface, cfg.VPN.Name)))
		return
	}
	emitResult(map[string]any{"logged_out": true})
}

func cmdSpeed(cfg *config.Config) {
	_ = cfg
	st := network.NewSpeedTest()
	result := st.Run()
	if result.Error != nil {
		emitOpError("speed", result.Error)
		return
	}
	emitResult(map[string]any{
		"download_mbps": result.DownloadMbps,
		"bytes":         result.BytesDownload,
		"duration_s":    result.Duration.Seconds(),
		"server":        result.ServerURL,
		"display":       result.FormatSpeed(),
	})
}

func cmdHotspot(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "hotspot requires an action (start|stop|status|clients)\n")
		emitError("invalid_input", "hotspot requires an action (start|stop|status|clients)")
		return
	}
	rejectSubcommandFlags("hotspot", args[1:])

	action := args[0]
	cap := hotspot.NewCreateAP(&hotspot.Config{
		SSID:       cfg.Hotspot.SSID,
		Passphrase: cfg.Hotspot.Passphrase,
		Channel:    cfg.Hotspot.Channel,
		FreqBand:   cfg.Hotspot.FreqBand,
		Method:     cfg.Hotspot.Method,
		Gateway:    cfg.Hotspot.Gateway,
	})

	switch action {
	case "start":
		if err := cap.Start(); err != nil {
			emitOpError("hotspot start", err)
			return
		}
		emitResult(map[string]any{"action": "start", "started": true})

	case "stop":
		if err := cap.Stop(); err != nil {
			emitOpError("hotspot stop", err)
			return
		}
		emitResult(map[string]any{"action": "stop", "stopped": true})

	case "status":
		status, err := cap.Status()
		if err != nil {
			emitOpError("hotspot status", err)
			return
		}
		emitResult(map[string]any{
			"active":       status.Active,
			"ssid":         status.SSID,
			"ip":           status.IP,
			"clients":      status.Clients,
			"ap_interface": status.APInterface,
		})

	case "clients":
		cm := hotspot.NewClientManager("ap0")
		clients, err := cm.ListClients()
		if err != nil {
			emitOpError("hotspot clients", err)
			return
		}
		out := make([]map[string]any, 0, len(clients))
		for _, c := range clients {
			name := c.Name
			if name == "" {
				name = c.MAC
			}
			out = append(out, map[string]any{"ip": c.IP, "mac": c.MAC, "name": name})
		}
		emitResult(map[string]any{"clients": out, "count": len(out)})

	default:
		fmt.Fprintf(os.Stderr, "unknown hotspot action: %s\n", action)
		emitError("invalid_input", fmt.Sprintf("unknown hotspot action: %s", action))
	}
}

func cmdNAT(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "nat requires an action (setup|cleanup|status)\n")
		emitError("invalid_input", "nat requires an action (setup|cleanup|status)")
		return
	}
	rejectSubcommandFlags("nat", args[1:])

	action := args[0]
	n := hotspot.NewNAT(
		cfg.Hotspot.Subnet,
		"ap0",
		cfg.VPN.Interface,
		cfg.Network.Interface,
	)

	switch action {
	case "setup":
		if err := n.Setup(); err != nil {
			emitOpError("nat setup", err)
			return
		}
		emitResult(map[string]any{"action": "setup", "configured": true})

	case "cleanup":
		n.Cleanup()
		emitResult(map[string]any{"action": "cleanup", "cleaned": true})

	case "status":
		status, err := n.Status()
		if err != nil {
			emitOpError("nat status", err)
			return
		}
		emitResult(map[string]any{
			"ip_forward":    status.IPForward,
			"nat_rules":     status.NATRules,
			"forward_rules": status.ForwardRules,
		})

	default:
		fmt.Fprintf(os.Stderr, "unknown nat action: %s\n", action)
		emitError("invalid_input", fmt.Sprintf("unknown nat action: %s", action))
	}
}

func cmdVPN(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "vpn requires an action (status|connect|disconnect|toggle|list|import)\n")
		emitError("invalid_input", fmt.Sprintf("vpn requires an action: %v", args))
		return
	}
	rejectSubcommandFlags("vpn", args[1:])

	action := args[0]
	m := vpn.NewManager(cfg.VPN.Interface, cfg.VPN.Name)

	switch action {
	case "status":
		status, err := m.Status()
		if err != nil {
			emitOpError("vpn status", err)
			return
		}
		profiles := status.Profiles
		if profiles == nil {
			profiles = make([]vpn.Profile, 0)
		}
		emitResult(map[string]any{
			"connected": status.Connected,
			"ip":        status.IP,
			"name":      status.ConnectionName,
			"interface": status.Interface,
			"profiles":  profiles,
		})

	case "list":
		if len(args) > 1 {
			emitError("invalid_input", fmt.Sprintf("unexpected vpn list args: %v", args[1:]))
			return
		}
		profiles, err := m.ListProfiles()
		if err != nil {
			emitOpError("vpn list", err)
			return
		}
		if profiles == nil {
			profiles = make([]vpn.Profile, 0)
		}
		emitResult(map[string]any{"profiles": profiles, "count": len(profiles)})

	case "connect":
		if len(args) > 2 {
			emitError("invalid_input", fmt.Sprintf("unexpected vpn connect args: %v", args[1:]))
			return
		}
		target := cfg.VPN.Name
		if len(args) == 2 {
			target = args[1]
			if target == "" {
				emitError("invalid_input", "vpn connect requires a non-empty profile name")
				return
			}
			if profiles, err := m.ListProfiles(); err == nil && len(profiles) > 0 {
				found := false
				available := make([]string, 0, len(profiles))
				for _, p := range profiles {
					available = append(available, p.Name)
					if p.Name == target {
						found = true
					}
				}
				if !found {
					emitError("invalid_input", fmt.Sprintf("unknown VPN profile %q (available: %s)", target, strings.Join(available, ", ")))
					return
				}
			}
		}
		if err := m.ConnectTo(target); err != nil {
			if strings.Contains(err.Error(), "unknown VPN profile") {
				emitError("invalid_input", err.Error())
				return
			}
			emitOpError("vpn connect", err)
			return
		}
		emitResult(map[string]any{"action": "connect", "connected": true, "name": target})

	case "import":
		if len(args) != 2 {
			emitError("invalid_input", fmt.Sprintf("vpn import requires one file argument: %v", args))
			return
		}
		name, err := m.Import(args[1])
		if err != nil {
			emitOpError("vpn import", err)
			return
		}
		emitResult(map[string]any{"action": "import", "name": name, "file": args[1]})

	case "disconnect":
		if len(args) > 2 {
			emitError("invalid_input", fmt.Sprintf("unexpected vpn disconnect args: %v", args[1:]))
			return
		}
		target := ""
		if len(args) == 2 {
			target = args[1]
			if target == "" {
				emitError("invalid_input", "vpn disconnect requires a non-empty profile name")
				return
			}
			if err := m.DisconnectTo(target); err != nil {
				if strings.Contains(err.Error(), "unknown VPN profile") {
					emitError("invalid_input", err.Error())
					return
				}
				emitOpError("vpn disconnect", err)
				return
			}
		} else {
			if err := m.Disconnect(); err != nil {
				emitOpError("vpn disconnect", err)
				return
			}
		}
		emitResult(map[string]any{"action": "disconnect", "connected": false, "name": target})

	case "toggle":
		connected, err := m.Toggle()
		if err != nil {
			emitOpError("vpn toggle", err)
			return
		}
		emitResult(map[string]any{"action": "toggle", "connected": connected})

	default:
		fmt.Fprintf(os.Stderr, "unknown vpn action: %s\n", action)
		emitError("invalid_input", fmt.Sprintf("unknown vpn action: %s", action))
	}
}
