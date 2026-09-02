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
- `conecta-cli` installed (see main project README)
- `create_ap` for hotspot functionality
- WireGuard for VPN (optional)

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
- **Auto-reconnect** - Automatically reconnect on disconnect

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
3. Check permissions: some commands need sudo

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
