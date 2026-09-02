package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"conecta/pkg/hotspot"
)

func main() {
	m := NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	go pollLoop(p)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func pollLoop(p *tea.Program) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		p.Send(refreshMsg{})
		<-ticker.C
	}
}

// ─── Model ───────────────────────────────────────────────────────────────────

type Model struct {
	hotspot    *hotspot.CreateAP
	status     *hotspot.Status
	clients    []hotspot.Client
	updatedAt  time.Time
	statusMsg  string
	statusExp  time.Time
}

func NewModel() Model {
	return Model{
		hotspot: hotspot.NewCreateAP(nil),
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

type refreshMsg struct{}
type actionResultMsg struct {
	success bool
	message string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		m = m.refresh()
	case actionResultMsg:
		m.statusMsg = msg.message
		m.statusExp = time.Now().Add(3 * time.Second)
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			return m, m.startHotspot()
		case "S":
			return m, m.stopHotspot()
		case "n":
			return m, m.setupNAT()
		case "r":
			return m, func() tea.Msg { return refreshMsg{} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	if m.statusMsg != "" && time.Now().After(m.statusExp) {
		m.statusMsg = ""
	}

	return m, nil
}

func (m Model) refresh() Model {
	s, _ := m.hotspot.Status()
	m.status = s
	cm := hotspot.NewClientManager("ap0")
	m.clients, _ = cm.ListClients()
	m.updatedAt = time.Now()
	return m
}

func (m Model) startHotspot() tea.Cmd {
	return func() tea.Msg {
		err := m.hotspot.Start()
		if err != nil {
			return actionResultMsg{false, "Error: " + err.Error()}
		}
		return actionResultMsg{true, "Hotspot started"}
	}
}

func (m Model) stopHotspot() tea.Cmd {
	return func() tea.Msg {
		err := m.hotspot.Stop()
		if err != nil {
			return actionResultMsg{false, "Error: " + err.Error()}
		}
		return actionResultMsg{true, "Hotspot stopped"}
	}
}

func (m Model) setupNAT() tea.Cmd {
	return func() tea.Msg {
		n := hotspot.NewNAT("192.168.12.0/24", "ap0", "wg0", "enp3s0")
		err := n.Setup()
		if err != nil {
			return actionResultMsg{false, "NAT error: " + err.Error()}
		}
		return actionResultMsg{true, "NAT configured"}
	}
}

// ─── View ────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	s := "📡 Conecta Hotspot  @ " + m.updatedAt.Format("15:04:05") + "\n\n"

	// Hotspot status
	if m.status != nil {
		status := "○ STOPPED"
		if m.status.Active {
			status = "● ACTIVE"
		}
		s += "═══ HOTSPOT ═══\n"
		s += "  Status: " + status + "\n"
		s += "  SSID: " + m.status.SSID + "\n"
		s += fmt.Sprintf("  Channel: %d / %s\n", m.status.Channel, m.status.FreqBand)
		s += "  IP: " + m.status.IP + "\n"
		s += fmt.Sprintf("  Clients: %d\n", m.status.Clients)
	} else {
		s += "Loading...\n"
	}

	// Clients
	s += "\n═══ CLIENTS (%d) ═══\n"
	if len(m.clients) == 0 {
		s += "  (none)\n"
	} else {
		s += fmt.Sprintf("  %-16s %-18s %s\n", "IP", "MAC", "NAME")
		for _, c := range m.clients {
			name := c.Name
			if name == "" {
				name = c.MAC
			}
			s += fmt.Sprintf("  %-16s %-18s %s\n", c.IP, c.MAC, name)
		}
	}

	// Status
	if m.statusMsg != "" {
		s += "\n  → " + m.statusMsg
	}

	// Footer
	s += "\n\n[s]start  [S]stop  [n]NAT  [r]refresh  [q]quit"

	return s
}
