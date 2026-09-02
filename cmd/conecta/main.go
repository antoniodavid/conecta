package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

	// Create model
	m := NewModel(cfg)

	// Run TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ─── Model ───────────────────────────────────────────────────────────────────

type Tab int

const (
	TabDashboard Tab = iota
	TabHotspot
	TabVPN
	TabLog
)

type Model struct {
	cfg      *config.Config
	tab      Tab
	quitting bool

	// Network state
	portal    *network.Portal
	conn      *network.Connection
	lastCheck time.Time

	// Hotspot state
	hotspot      *hotspot.CreateAP
	hotspotStatus *hotspot.Status
	clients      []hotspot.Client

	// VPN state
	vpn          *vpn.Manager
	vpnConnected bool

	// UI state
	logs   []string
	statusMsg string
	statusExp time.Time
}

func NewModel(cfg *config.Config) Model {
	m := Model{
		cfg:     cfg,
		portal:  network.NewPortal(&network.NetworkConfig{
			Gateway:   cfg.Network.Gateway,
			Interface: cfg.Network.Interface,
			PortalURL: cfg.Network.PortalURL,
		}),
		hotspot: hotspot.NewCreateAP(&hotspot.Config{
			SSID:       cfg.Hotspot.SSID,
			Passphrase: cfg.Hotspot.Passphrase,
			Channel:    cfg.Hotspot.Channel,
			FreqBand:   cfg.Hotspot.FreqBand,
			Method:     cfg.Hotspot.Method,
			Gateway:    cfg.Hotspot.Gateway,
		}),
		vpn: vpn.NewManager(cfg.VPN.Interface, cfg.VPN.Name),
	}

	// Auto-login if credentials are configured
	if cfg.Credentials.Username != "" && cfg.Credentials.Password != "" {
		m.addLog("Credentials loaded, auto-login enabled")
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.checkNetwork(),
		m.checkVPN(),
		tea.Tick(time.Duration(m.cfg.UI.RefreshSec)*time.Second, func(time.Time) tea.Msg {
			return refreshTickMsg{}
		}),
	)
}

// ─── Messages ────────────────────────────────────────────────────────────────

type refreshTickMsg struct{}
type networkCheckMsg struct {
	conn *network.Connection
	err  error
}
type hotspotStatusMsg struct {
	status *hotspot.Status
	err    error
}
type vpnStatusMsg struct {
	status *vpn.Status
	err    error
}
type actionResultMsg struct {
	success bool
	message string
}

// ─── Update ──────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case refreshTickMsg:
		return m, tea.Batch(
			m.checkNetwork(),
			m.checkVPN(),
			tea.Tick(time.Duration(m.cfg.UI.RefreshSec)*time.Second, func(time.Time) tea.Msg {
				return refreshTickMsg{}
			}),
		)

	case networkCheckMsg:
		m.conn = msg.conn
		m.lastCheck = time.Now()
		if msg.err != nil {
			m.addLog("Network error: " + msg.err.Error())
		}
		return m, nil

	case vpnStatusMsg:
		if msg.status != nil {
			m.vpnConnected = msg.status.Connected
		}
		if msg.err != nil {
			m.addLog("VPN error: " + msg.err.Error())
		}
		return m, nil

	case hotspotStatusMsg:
		m.hotspotStatus = msg.status
		if msg.err != nil {
			m.addLog("Hotspot error: " + msg.err.Error())
		}
		return m, nil

	case actionResultMsg:
		m.statusMsg = msg.message
		m.statusExp = time.Now().Add(3 * time.Second)
		if msg.success {
			m.addLog("✓ " + msg.message)
		} else {
			m.addLog("✗ " + msg.message)
		}
		return m, nil

	case tea.WindowSizeMsg:
		return m, nil
	}

	// Clear status message
	if m.statusMsg != "" && time.Now().After(m.statusExp) {
		m.statusMsg = ""
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "1":
		m.tab = TabDashboard
	case "2":
		m.tab = TabHotspot
	case "3":
		m.tab = TabVPN
	case "4":
		m.tab = TabLog

	case "s":
		if m.tab == TabHotspot {
			return m, m.startHotspot()
		}
	case "S":
		if m.tab == TabHotspot {
			return m, m.stopHotspot()
		}
	case "v":
		if m.tab == TabVPN {
			return m, m.toggleVPN()
		}
	case "n":
		if m.tab == TabHotspot {
			return m, m.setupNAT()
		}
	case "r":
		return m, m.checkNetwork()
	}

	return m, nil
}

// ─── Commands ────────────────────────────────────────────────────────────────

func (m Model) checkNetwork() tea.Cmd {
	return func() tea.Msg {
		conn, err := m.portal.CheckPortal()
		return networkCheckMsg{conn: conn, err: err}
	}
}

func (m Model) checkVPN() tea.Cmd {
	return func() tea.Msg {
		status, err := m.vpn.Status()
		return vpnStatusMsg{status: status, err: err}
	}
}

func (m Model) startHotspot() tea.Cmd {
	return func() tea.Msg {
		err := m.hotspot.Start()
		if err != nil {
			return actionResultMsg{success: false, message: "Failed to start: " + err.Error()}
		}
		return actionResultMsg{success: true, message: "Hotspot started"}
	}
}

func (m Model) stopHotspot() tea.Cmd {
	return func() tea.Msg {
		err := m.hotspot.Stop()
		if err != nil {
			return actionResultMsg{success: false, message: "Failed to stop: " + err.Error()}
		}
		return actionResultMsg{success: true, message: "Hotspot stopped"}
	}
}

func (m Model) toggleVPN() tea.Cmd {
	return func() tea.Msg {
		connected, err := m.vpn.Toggle()
		if err != nil {
			return actionResultMsg{success: false, message: "VPN error: " + err.Error()}
		}
		if connected {
			return actionResultMsg{success: true, message: "VPN connected"}
		}
		return actionResultMsg{success: true, message: "VPN disconnected"}
	}
}

func (m Model) setupNAT() tea.Cmd {
	return func() tea.Msg {
		n := hotspot.NewNAT(
			m.cfg.Hotspot.Subnet,
			"ap0",
			m.cfg.VPN.Interface,
			m.cfg.Network.Interface,
		)
		err := n.Setup()
		if err != nil {
			return actionResultMsg{success: false, message: "Failed to setup NAT: " + err.Error()}
		}
		return actionResultMsg{success: true, message: "NAT configured"}
	}
}

func (m *Model) addLog(msg string) {
	ts := time.Now().Format("15:04:05")
	m.logs = append(m.logs, fmt.Sprintf("%s %s", ts, msg))
	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}
}

// ─── View ────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.quitting {
		return "\n  ¡Hasta luego! 👋\n"
	}

	var s string

	// Header
	s += renderHeader(m)
	s += "\n\n"

	// Tab content
	switch m.tab {
	case TabDashboard:
		s += renderDashboard(m)
	case TabHotspot:
		s += renderHotspotTab(m)
	case TabVPN:
		s += renderVPNTab(m)
	case TabLog:
		s += renderLogTab(m)
	}

	// Status message
	if m.statusMsg != "" {
		s += "\n\n  → " + m.statusMsg
	}

	// Footer
	s += "\n\n"
	s += renderFooter(m)

	return s
}

func renderHeader(m Model) string {
	title := "📡 Conecta"
	tabs := "  [1]Dashboard  [2]Hotspot  [3]VPN  [4]Log"
	return title + tabs
}

func renderDashboard(m Model) string {
	s := "═══ Network Status ═══\n\n"

	if m.conn != nil {
		s += fmt.Sprintf("  Status: %s\n", m.conn.Status)
		s += fmt.Sprintf("  Gateway: %s\n", m.conn.Gateway)
		s += fmt.Sprintf("  Interface: %s\n", m.conn.Interface)
	} else {
		s += "  Checking...\n"
	}

	return s
}

func renderHotspotTab(m Model) string {
	s := "═══ Hotspot ═══\n\n"

	if m.hotspotStatus != nil {
		s += fmt.Sprintf("  Active: %v\n", m.hotspotStatus.Active)
		s += fmt.Sprintf("  SSID: %s\n", m.hotspotStatus.SSID)
		s += fmt.Sprintf("  IP: %s\n", m.hotspotStatus.IP)
		s += fmt.Sprintf("  Clients: %d\n", m.hotspotStatus.Clients)
	} else {
		s += "  Checking...\n"
	}

	s += "\n  [s]start  [S]stop  [n]NAT  [r]refresh"
	return s
}

func renderVPNTab(m Model) string {
	s := "═══ VPN ═══\n\n"
	s += fmt.Sprintf("  Connected: %v\n", m.vpnConnected)
	s += "\n  [v]toggle"
	return s
}

func renderLogTab(m Model) string {
	s := "═══ Log ═══\n\n"

	start := 0
	if len(m.logs) > 20 {
		start = len(m.logs) - 20
	}

	for _, line := range m.logs[start:] {
		s += "  " + line + "\n"
	}

	if len(m.logs) == 0 {
		s += "  (empty)\n"
	}

	return s
}

func renderFooter(m Model) string {
	return "[q]quit"
}
