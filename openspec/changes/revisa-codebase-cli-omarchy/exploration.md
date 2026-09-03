## Exploration: CLI and Omarchy plugin review

### Current State

`conecta` is a single Go 1.24 module with three binaries (`conecta-cli`, `conecta`, and `hotspot`) and reusable packages for portal access, hotspot management, NAT, VPN, configuration, and speed testing. The Omarchy integration is a Quickshell QML bar widget backed by four Bash scripts that call `conecta-cli` and inspect host networking directly.

The repository currently has local, pre-existing modifications in `Panel.qml`, the status script, `pkg/network/portal.go`, the compiled CLI, and the icon. Findings describe the current worktree and are not attributed to this exploration.

Checks performed:

- `go test -cover ./...` — passed, 7 packages / 29 tests.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `bash -n omarchy-plugin/bin/*` — passed.
- `qmllint omarchy-plugin/Panel.qml` — passed with no diagnostics.
- On this host, `iw dev` emits `Interface wlo1` on one line; the current detector expects a line containing only `Interface`.
- With `CONNECTA_CLI=/bin/false`, the login, hotspot-start, and VPN-toggle scripts exit 1 with empty output.

### Confirmed Findings

#### Critical functional defects

- `pkg/hotspot/create_ap.go:181-195` — `detectWiFiInterface` looks for `strings.TrimSpace(line) == "Interface"` and then reads the next line. Real `iw dev` output is `Interface <name>` on the same line, so hotspot start reports no Wi-Fi interface and stops before starting `create_ap`.
- `cmd/cli/main.go:23-24,83-87` — the global `flag` set is parsed before the command is selected, and `cmdLogin` ignores its `args` parameter and parses the global set again. The documented `conecta-cli login --user ... --pass ...` form therefore does not reliably bind subcommand credentials.
- `cmd/conecta/main.go:99-105,116-166,231-243` — `hotspotStatusMsg` is handled but no command ever produces it. The main TUI never calls `CreateAP.Status`, so the hotspot tab remains `Checking...` and refreshes do not update it.
- `cmd/conecta/main.go:91-94` — the “auto-login” path only logs that credentials are loaded; it never calls `Portal.Login`.
- `cmd/hotspot/main.go:44-47,118-126` — the standalone hotspot TUI does not load `~/.config/conecta/config.yaml`; it uses package defaults and hardcoded NAT interfaces/subnet, ignoring user configuration.
- `cmd/hotspot/main.go:151` — the view appends `"CLIENTS (%d)"` without `fmt.Sprintf`, so the literal `%d` is rendered.
- `omarchy-plugin/bin/omarchy-conecta-login:15-21`, `omarchy-plugin/bin/omarchy-conecta-hotspot:14-21`, `omarchy-plugin/bin/omarchy-conecta-vpn:23-29` — `set -e` exits immediately when the CLI fails, before the intended failure JSON is emitted. The controlled failing-CLI check confirmed exit 1 with empty output for all three action paths.
- `omarchy-plugin/Panel.qml:552-556` — the speed-test control invokes `runLogin("speed")`; its raw CLI output is not displayed or stored, and the process callback only refreshes status. The feature is effectively a no-op from the user’s perspective.

#### High-risk integration and correctness defects

- `omarchy-plugin/Panel.qml:72-84` and `omarchy-plugin/bin/omarchy-conecta-status:80-101` — the QML status handler calls `JSON.parse(data)` on `SplitParser` output, while the script emits pretty-printed multi-line JSON. Unless the parser is configured to aggregate the complete document, each line is not a JSON document and status remains unset. This needs one runtime fixture test because it depends on the Quickshell parser contract.
- `omarchy-plugin/bin/omarchy-conecta-status:46-56` — the signal value removes `-` before comparing against negative dBm thresholds. Positive values then satisfy the `>= -90` branch and produce zero signal, so Wi-Fi strength is wrong.
- `omarchy-plugin/bin/omarchy-conecta-status:74-78` and `omarchy-plugin/bin/omarchy-conecta-vpn:31-42` — VPN status is hardcoded to `wg0`, ignoring the configured VPN interface used by the Go CLI.
- `omarchy-plugin/bin/omarchy-conecta-status:81-100` — JSON fields are interpolated without JSON escaping. A network-provided SSID or a configured value containing quotes, backslashes, or newlines can make the status payload invalid.
- `pkg/hotspot/create_ap.go:34-44,144` — hotspot setup writes `/etc/create_ap.conf` as the unprivileged process, then uses `sudo` only for `systemctl`. Normal CLI/plugin execution can fail before service startup, and a GUI-launched `sudo` may have no interactive password path.
- `pkg/hotspot/create_ap.go:26-36` — startup kills `hostapd`/`dnsmasq` and disables managed Wi-Fi before configuration is written. If configuration or interface detection fails, there is no rollback and the host can be left in a degraded network state.
- `pkg/network/portal.go:32` and `pkg/network/speed.go:21` — TLS certificate verification is disabled globally for portal and speed-test HTTP clients.
- `pkg/network/portal.go:158,188` — `http.NewRequest` errors are ignored; a malformed configured URL can leave `req` nil and cause a panic when headers are assigned.
- `pkg/network/portal.go:250-257` — an unrecognized login response defaults to `PortalConnected`, creating false-positive authentication.

#### Medium/low correctness and maintainability issues

- `pkg/network/speed.go:55-80` — HTTP status codes are not checked, partial reads with non-EOF errors can be reported as successful tests, and an empty URL list formats a `%!w(<nil>)` error.
- `pkg/hotspot/clients.go:81-104` — client parsing assumes every `Station ` block has a non-empty first field; malformed command output can panic. It also only recognizes uppercase `RX bytes:`/`TX bytes:`, while typical `iw` output uses lowercase labels.
- `pkg/hotspot/clients.go:35-61` and `pkg/hotspot/create_ap.go:61-79` — missing system tools/errors are converted to empty status rather than surfaced, making “tool unavailable” indistinguishable from “nothing connected.”
- `pkg/hotspot/nat.go:70-78` — cleanup chooses the currently existing VPN interface, so rules created while the VPN was up may remain if the VPN is down before cleanup.
- `pkg/hotspot/nat.go:90-100` — NAT status counts all system `MASQUERADE`/`ACCEPT` rules rather than Conecta’s rules.
- `cmd/conecta/main.go:136-143,169-177` — several handled messages return before the status-expiration block at lines 183-186, so transient action messages can remain visible indefinitely.
- `omarchy-plugin/Panel.qml:41-44,564-569` and `omarchy-plugin/manifest.json:64-67` — `autoReconnect` is declared in the manifest but never read or implemented.
- `omarchy-plugin/Panel.qml:48-52` — the hotspot-active icon string contains corrupted/non-icon text (`�dığında`).
- `pkg/config/config.go:104-116` — saving a config containing credentials uses mode `0644`, making the file world-readable.
- `MANUAL.md:93-96` — documentation contains credential-looking plaintext values. If they are real, they must be revoked; if examples, replace them with unmistakable placeholders.

### Affected Areas

- `cmd/cli/main.go` — command dispatch and subcommand flag parsing.
- `cmd/conecta/main.go` — main Bubble Tea model, polling, actions, and rendering.
- `cmd/hotspot/main.go` — standalone hotspot TUI and hardcoded configuration.
- `pkg/hotspot/create_ap.go` — Wi-Fi detection, privileged configuration, lifecycle, and status.
- `pkg/hotspot/clients.go` — `iw`, DHCP lease, and ARP client discovery.
- `pkg/hotspot/nat.go` — forwarding/NAT setup, cleanup, and status.
- `pkg/network/portal.go` — portal HTTP/TLS/login response handling.
- `pkg/network/speed.go` — download test validation and error handling.
- `omarchy-plugin/Panel.qml` — QML process lifecycle, status parsing, controls, and settings.
- `omarchy-plugin/bin/omarchy-conecta-{status,login,hotspot,vpn}` — shell-to-CLI contract, host inspection, and failure propagation.
- `omarchy-plugin/manifest.json` — plugin settings and activation metadata.
- `README.md`, `MANUAL.md`, `omarchy-plugin/README.md` — installation, privilege, configuration, and credential documentation.

### Approaches

1. **Minimal repair with a stable JSON CLI contract** — fix command parsing and Wi-Fi detection, add machine-readable status/action results to `conecta-cli`, make the plugin scripts thin adapters, and add focused tests around parsers and process failures.
   - Pros: smallest useful scope; removes duplicated shell parsing and fixes the current plugin path without a daemon.
   - Cons: privileged operations still need an explicit, non-interactive authorization/install model.
   - Effort: Medium.

2. **Shared user-level service/IPC** — move polling and privileged operations behind a long-lived service and let both TUIs and the plugin consume one API.
   - Pros: better asynchronous errors, one source of truth, and cleaner privilege boundaries.
   - Cons: adds service lifecycle, IPC, installation, and rollback complexity before the current defects are fixed.
   - Effort: High.

### Recommendation

Choose Approach 1 first. Treat the CLI output and exit status as the contract, keep the plugin as a thin consumer, and resolve the privilege model before exposing hotspot/NAT/VPN actions from Quickshell. The first proposal should be split into: (a) CLI/core correctness and test seams, (b) plugin JSON/process integration, and (c) installation/privilege documentation. Do not introduce a daemon until the minimal contract cannot meet the required interaction model.

### Risks

- Current local modifications make it unsafe to assume every finding belongs to the base revision; review the diff before implementation.
- Hotspot, NAT, VPN, and portal behavior depends on host services and permissions that unit tests cannot prove.
- A plugin runtime fixture is required to confirm the exact `SplitParser` behavior and Omarchy manifest/entry-point contract.
- Removing TLS verification may have been intended as a captive-portal workaround, but retaining it exposes credentials and speed-test traffic to interception.
- Privileged commands launched from a bar widget can hang or fail silently without a documented authorization path.

### Ready for Proposal

Yes. The problem is sufficiently understood for a proposal, provided the proposal keeps the first slice focused on the confirmed CLI/plugin contract defects and explicitly records the privilege and runtime-fixture decisions.
