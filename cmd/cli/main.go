package main

import (
	"flag"
	"fmt"
	"os"

	"conecta/pkg/config"
	"conecta/pkg/hotspot"
	"conecta/pkg/network"
	"conecta/pkg/vpn"
)

func main() {
	// Load config
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Parse flags
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	args := flag.Args()
	command := args[0]

	switch command {
	case "help", "--help", "-h":
		printUsage()
	case "status":
		cmdStatus(cfg)
	case "login":
		cmdLogin(cfg, args[1:])
	case "logout":
		cmdLogout(cfg)
	case "speed":
		cmdSpeed(cfg)
	case "hotspot":
		cmdHotspot(cfg, args[1:])
	case "nat":
		cmdNAT(cfg, args[1:])
	case "vpn":
		cmdVPN(cfg, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(helpText)
}

func cmdStatus(cfg *config.Config) {
	portal := network.NewPortal(&network.NetworkConfig{
		Gateway:   cfg.Network.Gateway,
		Interface: cfg.Network.Interface,
		PortalURL: cfg.Network.PortalURL,
	})

	conn, err := portal.CheckPortal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking portal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", conn.Status)
	fmt.Printf("Gateway: %s\n", conn.Gateway)
	fmt.Printf("Interface: %s\n", conn.Interface)
	if conn.LastError != nil {
		fmt.Printf("Error: %v\n", conn.LastError)
	}
}

func cmdLogin(cfg *config.Config, args []string) {
	var user, pass string
	flag.StringVar(&user, "user", "", "Username")
	flag.StringVar(&pass, "pass", "", "Password")
	flag.Parse()

	// Use config credentials if not provided via flags
	if user == "" {
		user = cfg.Credentials.Username
	}
	if pass == "" {
		pass = cfg.Credentials.Password
	}

	if user == "" || pass == "" {
		fmt.Fprintf(os.Stderr, "Error: --user and --pass are required (or set in config)\n")
		os.Exit(1)
	}

	portal := network.NewPortal(&network.NetworkConfig{
		Gateway:   cfg.Network.Gateway,
		Interface: cfg.Network.Interface,
		PortalURL: cfg.Network.PortalURL,
	})

	conn, err := portal.Login(user, pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Login successful: %s\n", conn.Status)
}

func cmdLogout(cfg *config.Config) {
	portal := network.NewPortal(&network.NetworkConfig{
		Gateway:   cfg.Network.Gateway,
		Interface: cfg.Network.Interface,
		PortalURL: cfg.Network.PortalURL,
	})

	if err := portal.Logout(); err != nil {
		fmt.Fprintf(os.Stderr, "Logout failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Logged out successfully")
}

func cmdSpeed(cfg *config.Config) {
	fmt.Println("Running speed test...")
	st := network.NewSpeedTest()
	result := st.Run()
	fmt.Println(result.FormatSpeed())
}

func cmdHotspot(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: hotspot requires an action (start|stop|status|clients)\n")
		os.Exit(1)
	}

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
			fmt.Fprintf(os.Stderr, "Failed to start hotspot: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Hotspot started")

	case "stop":
		if err := cap.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stop hotspot: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Hotspot stopped")

	case "status":
		status, err := cap.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Active: %v\n", status.Active)
		fmt.Printf("SSID: %s\n", status.SSID)
		fmt.Printf("IP: %s\n", status.IP)
		fmt.Printf("Clients: %d\n", status.Clients)

	case "clients":
		cm := hotspot.NewClientManager("ap0")
		clients, err := cm.ListClients()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list clients: %v\n", err)
			os.Exit(1)
		}
		if len(clients) == 0 {
			fmt.Println("No clients connected")
			return
		}
		fmt.Printf("Clients (%d):\n", len(clients))
		for _, c := range clients {
			name := c.Name
			if name == "" {
				name = c.MAC
			}
			fmt.Printf("  %s  %s  %s\n", c.IP, c.MAC, name)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown hotspot action: %s\n", action)
		os.Exit(1)
	}
}

func cmdNAT(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: nat requires an action (setup|cleanup|status)\n")
		os.Exit(1)
	}

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
			fmt.Fprintf(os.Stderr, "Failed to setup NAT: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("NAT configured")

	case "cleanup":
		n.Cleanup()
		fmt.Println("NAT cleaned up")

	case "status":
		status, err := n.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get NAT status: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("IP Forward: %v\n", status.IPForward)
		fmt.Printf("NAT Rules: %d\n", status.NATRules)
		fmt.Printf("Forward Rules: %d\n", status.ForwardRules)

	default:
		fmt.Fprintf(os.Stderr, "Unknown NAT action: %s\n", action)
		os.Exit(1)
	}
}

func cmdVPN(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: vpn requires an action (status|connect|disconnect|toggle)\n")
		os.Exit(1)
	}

	action := args[0]
	m := vpn.NewManager(cfg.VPN.Interface, cfg.VPN.Name)

	switch action {
	case "status":
		status, err := m.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get VPN status: %v\n", err)
			os.Exit(1)
		}
		if status.Connected {
			fmt.Printf("Connected: %v\n", status.Connected)
			fmt.Printf("IP: %s\n", status.IP)
			fmt.Printf("Connection: %s\n", status.ConnectionName)
		} else {
			fmt.Println("VPN: Disconnected")
		}

	case "connect":
		if err := m.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("VPN connected")

	case "disconnect":
		if err := m.Disconnect(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to disconnect: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("VPN disconnected")

	case "toggle":
		connected, err := m.Toggle()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to toggle: %v\n", err)
			os.Exit(1)
		}
		if connected {
			fmt.Println("VPN connected")
		} else {
			fmt.Println("VPN disconnected")
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown VPN action: %s\n", action)
		os.Exit(1)
	}
}
