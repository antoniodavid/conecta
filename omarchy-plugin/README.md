# Conecta - Omarchy Plugin

WiFi hotspot sharing and VPN control for Cuba's ETECSA network.

## Features

- 📡 **ETECSA Connection** - Monitor and manage your internet connection
- 🔥 **WiFi Hotspot** - Share your connection with other devices
- 🔒 **VPN Control** - Route traffic through WireGuard VPN
- 🚀 **Speed Test** - Measure your connection speed

## Installation

### Prerequisites

- Omarchy desktop environment
- `conecta-cli` installed and on `PATH` (see main project README;
  user-owned `~/.local/bin` or root-owned `/usr/local/bin`, never setuid)
- `jq` (adapters build and validate JSON with it)
- `create_ap` for hotspot functionality
- `iw` for WiFi detection, `iptables` for NAT, WireGuard for VPN (optional)

### Install Plugin

```bash
# Clone or copy plugin to Omarchy plugins directory
cd ~/.local/share/omarchy/plugins/
git clone <repo-url> conecta
# or
cp -r /path/to/conecta/omarchy-plugin ~/.local/share/omarchy/plugins/conecta
```

### Enable Plugin

```bash
omarchy plugin enable conecta.network
```

## Configuration

The plugin reads configuration from `~/.config/conecta/config.yaml`.

### Plugin Settings

Right-click the bar widget to configure:
- **Refresh interval** - How often to check status (default: 30s)
- **Show hotspot** - Show/hide hotspot controls
- **Show VPN** - Show/hide VPN controls
- **Show speed test** - Show/hide speed test button

## Authorization policy

A bar widget cannot answer a password prompt, so privileged actions are
gated: hotspot start/stop, NAT setup, and VPN connect/disconnect require a
pre-authorized non-interactive setup (e.g. a root-installed sudoers
NOPASSWD entry for the documented commands). Until that authorization is
resolved, those actions fail closed — the CLI exits `4` with parseable
failure JSON and the host is left untouched. Keep bar-triggered
hotspot/NAT/VPN disabled until the setup is authorized. Read-only status
and portal login need no privileges.

## Usage

### Bar Widget

The widget shows:
- 🔴 Disconnected
- 🟠 Authentication needed
- 🟢 Connected
- 📡 Hotspot active
- 🔒 VPN connected

Click the widget to open the control panel.

### Control Panel

From the popup panel you can:
- **Login/Logout** - Authenticate with ETECSA portal
- **Start/Stop Hotspot** - Toggle WiFi sharing
- **Connect/Disconnect VPN** - Toggle VPN connection
- **Speed Test** - Measure download speed

### CLI Commands

You can also use the CLI directly:

```bash
# Check status
conecta-cli status

# Login
conecta-cli login

# Start hotspot
conecta-cli hotspot start

# Connect VPN
conecta-cli vpn connect
```

## Architecture

```
omarchy-plugin/
├── manifest.json     # Plugin metadata
├── Panel.qml         # Bar widget UI
├── bin/
│   ├── omarchy-conecta-status    # Status script (JSON output)
│   ├── omarchy-conecta-hotspot   # Hotspot control
│   ├── omarchy-conecta-vpn       # VPN control
│   └── omarchy-conecta-login     # ETECSA login
└── assets/
    └── icon.svg      # Plugin icon
```

## Troubleshooting

### Widget not showing

1. Check plugin is enabled: `omarchy plugin list`
2. Check logs: `journalctl -f`
3. Verify scripts are executable: `ls -la ~/.local/share/omarchy/plugins/conecta/bin/`

### Commands not working

1. Verify `conecta-cli` is installed: `which conecta-cli`
2. Test scripts manually: `~/.local/share/omarchy/plugins/conecta/bin/omarchy-conecta-status`
3. Check config: `cat ~/.config/conecta/config.yaml`

### Hotspot not starting

1. Check WiFi interface: `iw dev`
2. Check create_ap: `systemctl status create_ap`
3. Check authorization: privileged actions need pre-authorized non-interactive
   sudo (`sudo -n true` must succeed); without it they exit `4` by design —
   see "Authorization policy" above

## Development

### Testing Scripts

```bash
# Test status output
./bin/omarchy-conecta-status

# Test hotspot
./bin/omarchy-conecta-hotspot status

# Test VPN
./bin/omarchy-conecta-vpn status
```

### Adding Features

1. Add new scripts to `bin/`
2. Update `Panel.qml` to expose new controls
3. Update `manifest.json` if adding new config options

## License

MIT
