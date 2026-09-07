package hotspot

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"conecta/pkg/network"
)

// CreateAP manages the create_ap service
type CreateAP struct {
	config *Config
}

// NewCreateAP creates a new create_ap manager
func NewCreateAP(config *Config) *CreateAP {
	if config == nil {
		config = DefaultConfig()
	}
	return &CreateAP{config: config}
}

// Start starts the hotspot
func (c *CreateAP) Start() (err error) {
	// Privilege pre-check before any destructive step (fail closed, exit 4 upstream).
	if err := checkAuthz(); err != nil {
		return err
	}
	// Kill any stale processes
	exec.Command("sudo", "killall", "hostapd").Run()
	exec.Command("sudo", "killall", "dnsmasq").Run()

	// Disable WiFi managed mode. From here on, any failure must restore the
	// radio so it is never left switched off.
	exec.Command("sudo", "nmcli", "r", "wifi", "off").Run()
	exec.Command("sudo", "rfkill", "unblock", "wlan").Run()
	defer func() {
		if err != nil {
			restoreRadio()
		}
	}()

	// Write config
	if err := c.writeConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Start service
	cmd := exec.Command("sudo", "systemctl", "start", "create_ap")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start create_ap: %w", err)
	}

	return nil
}

// Stop stops the hotspot
func (c *CreateAP) Stop() error {
	if err := checkAuthz(); err != nil {
		return err
	}
	cmd := exec.Command("sudo", "systemctl", "stop", "create_ap")
	stopErr := cmd.Run()

	// Always restore the radio, even when the stop failed, so WiFi is never
	// left switched off. restoreRadio ignores its own error (best-effort).
	restoreRadio()

	if stopErr != nil {
		return fmt.Errorf("failed to stop create_ap: %w", stopErr)
	}
	return nil
}

// restoreRadio re-enables the WiFi radio via nmcli. Errors are ignored:
// it is best-effort recovery and must not mask the primary error.
func restoreRadio() {
	exec.Command("sudo", "nmcli", "r", "wifi", "on").Run()
}

// Status returns the current hotspot status
func (c *CreateAP) Status() (*Status, error) {
	s := &Status{}

	// Check service
	out, _ := exec.Command("systemctl", "is-active", "create_ap").Output()
	s.Active = strings.TrimSpace(string(out)) == "active"

	// Check daemons
	s.Hostapd = isRunning("hostapd")
	s.Dnsmasq = isRunning("dnsmasq")

	// Read config
	c.readConfig(s)

	// Count clients always from the live AP interface (never a hardcoded
	// iface): the create_ap unit can be inactive while the AP iface persists.
	s.APInterface = apInterface()
	s.Clients = c.countClients()

	return s, nil
}

// IsRunning returns true if the hotspot is running
func (c *CreateAP) IsRunning() bool {
	out, err := exec.Command("systemctl", "is-active", "create_ap").Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

func (c *CreateAP) writeConfig() error {
	// Detect WiFi interface
	wifiIface, err := detectWiFiInterface()
	if err != nil {
		return err
	}

	// Determine internet interface
	internetIface := c.config.InternetInterface
	if internetIface == "" {
		internetIface = "enp3s0" // default
	}

	config := fmt.Sprintf(`CHANNEL=%s
GATEWAY=%s
WPA_VERSION=2
ETC_HOSTS=0
DHCP_DNS=gateway
NO_DNS=0
NO_DNSMASQ=0
HIDDEN=0
MAC_FILTER=0
MAC_FILTER_ACCEPT=/etc/hostapd/hostapd.accept
ISOLATE_CLIENTS=0
SHARE_METHOD=%s
IEEE80211N=1
IEEE80211AC=0
IEEE80211AX=0
HT_CAPAB=[HT40+]
VHT_CAPAB=
DRIVER=nl80211
NO_VIRT=0
COUNTRY=
FREQ_BAND=%s
NEW_MACADDR=
DAEMONIZE=0
DAEMON_PIDFILE=
DAEMON_LOGFILE=/dev/null
DNS_LOGFILE=
NO_HAVEGED=0
WIFI_IFACE=%s
INTERNET_IFACE=%s
SSID=%s
PASSPHRASE=%s
USE_PSK=0
ADDN_HOSTS=`,
		c.config.Channel,
		c.config.Gateway,
		c.config.Method,
		c.config.FreqBand,
		wifiIface,
		internetIface,
		c.config.SSID,
		c.config.Passphrase,
	)

	// Write config through sudo tee: /etc/create_ap.conf is root-owned.
	// The exact config bytes are fed via stdin, so sudo never sees them as
	// arguments (no quoting issues, no shell interpretation).
	cmd := exec.Command("sudo", "tee", "/etc/create_ap.conf")
	cmd.Stdin = strings.NewReader(config)
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("sudo tee /etc/create_ap.conf failed: %w (%s)", err, msg)
		}
		return fmt.Errorf("sudo tee /etc/create_ap.conf failed: %w", err)
	}
	return nil
}

func (c *CreateAP) readConfig(s *Status) {
	data, err := os.ReadFile("/etc/create_ap.conf")
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "SSID=") {
			s.SSID = strings.TrimPrefix(line, "SSID=")
		} else if strings.HasPrefix(line, "GATEWAY=") {
			s.IP = strings.TrimPrefix(line, "GATEWAY=")
		} else if strings.HasPrefix(line, "CHANNEL=") {
			ch := strings.TrimPrefix(line, "CHANNEL=")
			if ch != "default" {
				fmt.Sscanf(ch, "%d", &s.Channel)
			}
		} else if strings.HasPrefix(line, "FREQ_BAND=") {
			s.FreqBand = strings.TrimPrefix(line, "FREQ_BAND=")
		} else if strings.HasPrefix(line, "SHARE_METHOD=") {
			s.Method = strings.TrimPrefix(line, "SHARE_METHOD=")
		}
	}
}

// apInterface returns the name of the interface currently in AP mode via
// `iw dev`; empty when none is found (callers then fall back to "ap0").
func apInterface() string {
	out, err := exec.Command("iw", "dev").Output()
	if err != nil {
		return ""
	}
	return parseAPInterface(string(out))
}

// parseAPInterface parses `iw dev` output for the interface whose following
// type line is "AP", mirroring the network package's same-line Interface +
// following-type tracking. Unnamed/P2P-device blocks are skipped; no AP
// interface yields "". Names are kept verbatim.
func parseAPInterface(output string) string {
	var current string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "phy#") || strings.HasPrefix(trimmed, "Unnamed") {
			// A new device block starts: later type lines do not belong to
			// the previous interface.
			current = ""
			continue
		}
		if strings.HasPrefix(trimmed, "Interface ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				current = strings.Join(fields[1:], " ")
			} else {
				current = ""
			}
			continue
		}
		if current != "" && strings.HasPrefix(trimmed, "type ") {
			typ := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "type ")))
			if typ == "ap" {
				return current
			}
		}
	}
	return ""
}

func (c *CreateAP) countClients() int {
	iface := apInterface()
	if iface == "" {
		iface = "ap0"
	}
	out, err := exec.Command("iw", "dev", iface, "station", "dump").Output()
	if err != nil {
		return 0
	}
	return strings.Count(string(out), "Station ")
}

// ParseIWInterfaces parses `iw dev` output for same-line "Interface <name>" entries.
// It delegates to the network package so CLI status and hotspot share one
// parser, including the P2P-device skip. Hostile names are kept verbatim.
func ParseIWInterfaces(output string) ([]string, error) {
	ifaces := network.ParseWiFiInterfaces(output)
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no WiFi interface found")
	}
	return ifaces, nil
}

// checkAuthz verifies non-interactive privilege before destructive steps.
// It fails closed: denied or unavailable authz returns an authz error.
// The probe runs an actually-allowed command: the scoped sudoers drop-in
// (contrib/sudoers-conecta) does not permit plain `true`.
// Exit code 3 means the unit is inactive but sudo itself succeeded, so the
// probe is authorized; any other failure (1/126/127) means sudo denied or
// is unavailable.
func checkAuthz() error {
	_, err := exec.Command("sudo", "-n", "systemctl", "is-active", "create_ap").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 3 {
			return nil
		}
		return fmt.Errorf("authz: privileged action denied or unavailable (sudoers drop-in missing; run ./deploy.sh --setup-privileges): %w", err)
	}
	return nil
}

// detectWiFiInterface detects the WiFi interface via the shared network helper.
func detectWiFiInterface() (string, error) {
	return network.DetectWiFiInterface()
}

// isRunning checks if a process is running
func isRunning(name string) bool {
	out, err := exec.Command("pgrep", "-x", name).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}
