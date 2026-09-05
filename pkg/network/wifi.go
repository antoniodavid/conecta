package network

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var dbmRe = regexp.MustCompile(`signal:\s*(-?\d+(?:\.\d+)?)`)

// ParseSignalDBM extracts a signed dBm value from `iw dev <iface> link` output.
// The sign is preserved: -70 stays -70, never 70.
func ParseSignalDBM(line string) (int, error) {
	m := dbmRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0, fmt.Errorf("no signal dBm found in %q", line)
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid dBm %q: %w", m[1], err)
	}
	// Truncate toward zero but keep the sign (e.g. -45.5 -> -45).
	return int(f), nil
}

// DBMToPercent maps -30dBm (excellent) to 100 and -90dBm (unusable) to 0.
func DBMToPercent(dbm int) int {
	if dbm >= -30 {
		return 100
	}
	if dbm <= -90 {
		return 0
	}
	return (dbm + 90) * 100 / 60
}

// LinkSnapshot is the parsed state of one wireless link. Available is false
// when the tool is missing, the interface is missing, or nothing is
// connected — never a zero-signal success.
type LinkSnapshot struct {
	Available bool
	SSID      string
	SignalDBM int
	HasSignal bool
	Signal    int // 0-100 percent, valid only when HasSignal is true
}

// SnapshotLink runs `iw dev <iface> link` and parses SSID plus signed signal.
// It execs one fixed argv; network-provided strings stay data for JSON escaping.
func SnapshotLink(iface string) LinkSnapshot {
	var snap LinkSnapshot
	if iface == "" {
		return snap
	}
	out, err := exec.Command("iw", "dev", iface, "link").Output()
	if err != nil {
		return snap
	}
	text := string(out)
	if !strings.Contains(text, "Connected to") {
		return snap
	}
	snap.Available = true
	snap.SSID = ParseIWLinkSSID(text)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "signal:") {
			dbm, err := ParseSignalDBM(line)
			if err != nil {
				break
			}
			snap.SignalDBM = dbm
			snap.Signal = DBMToPercent(dbm)
			snap.HasSignal = true
			break
		}
	}
	return snap
}

// ParseWiFiInterfaces parses `iw dev` output for same-line "Interface <name>"
// entries, skipping P2P-device entries. The type line follows each interface
// block, so a block whose type contains P2P-device is dropped; managed and
// any other non-P2P type (or a missing type line) is accepted. Names are kept
// verbatim; callers must escape, never eval.
func ParseWiFiInterfaces(output string) []string {
	var names []string
	var isP2P []bool
	current := -1
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "phy#") || strings.HasPrefix(trimmed, "Unnamed") {
			// Close the current block: a later type line belongs to the
			// unnamed/P2P-device pseudo-device, not the previous interface.
			current = -1
			continue
		}
		if strings.HasPrefix(trimmed, "Interface ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				if name := strings.Join(fields[1:], " "); name != "" {
					names = append(names, name)
					isP2P = append(isP2P, false)
					current = len(names) - 1
				}
			}
			continue
		}
		if current >= 0 && strings.HasPrefix(trimmed, "type ") {
			typ := strings.TrimSpace(strings.TrimPrefix(trimmed, "type "))
			if strings.Contains(strings.ToLower(typ), "p2p-device") {
				isP2P[current] = true
			}
		}
	}
	var out []string
	for i, name := range names {
		if !isP2P[i] {
			out = append(out, name)
		}
	}
	return out
}

// DetectWiFiInterface runs fixed-argv `iw dev` and returns the first wireless
// interface, skipping P2P-device entries.
func DetectWiFiInterface() (string, error) {
	out, err := exec.Command("iw", "dev").Output()
	if err != nil {
		return "", fmt.Errorf("tool unavailable: iw: %w", err)
	}
	ifaces := ParseWiFiInterfaces(string(out))
	if len(ifaces) == 0 {
		return "", fmt.Errorf("no WiFi interface found")
	}
	return ifaces[0], nil
}

// ParseIWLinkSSID extracts the SSID verbatim from `iw` link output.
// It keeps hostile names intact; callers must JSON-escape, never eval.
func ParseIWLinkSSID(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SSID:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "SSID:"))
		}
	}
	return ""
}
