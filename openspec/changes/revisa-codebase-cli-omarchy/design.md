# Design: Make the Conecta CLI and Omarchy Plugin Dependable

## Technical Approach

Repair seams in place: no new binaries, packages, daemon, or IPC. `conecta-cli` becomes the single machine-readable boundary (one compact JSON document on stdout + deterministic exit codes). Bash adapters stop duplicating `iw`/`ip` parsing and stop swallowing failures; QML parses one complete document. The missing hotspot status command joins the existing Bubbletea tick loop.

## Architecture Decisions

| Decision | Options considered | Choice and rationale |
|----------|-------------------|----------------------|
| CLI dispatch | Global `flag` re-parse vs per-subcommand `FlagSet` | `FlagSet` per subcommand — global re-parse drops `login --user/--pass`. |
| Contract shape | Greppable text vs JSON envelope | Compact single-line `{ok,data,error}` on stdout, diagnostics on stderr — text breaks on hostile SSIDs. |
| Exit codes | `0/1` vs `0/2/3/4` | `0` ok, `2` input/config, `3` unavailable/op failure, `4` authz — spec-mandated; adapters propagate unchanged. |
| Status ownership | Shell `iw` parsing vs CLI-owned detection | CLI owns detection (same-line `Interface <name>`, signed dBm, configured VPN iface) — kills duplicated buggy parsers. |
| Privilege | GUI `sudo` vs pre-authorized install | Fail closed, exit `4`, no destructive setup — bar-launched `sudo` has no password path. |
| TLS | Skip-verify vs verify-by-default | Verify by default; captive-portal exception explicit and scoped; unknown logins fail closed. |

## Data Flow

```
Panel.qml Process ──→ bin/omarchy-conecta-* ──→ conecta-cli (JSON + exit N)
        │                    │ (verbatim passthrough, no re-parse)
        │                    └──────── failure JSON + same exit on error
        └── parse ONE complete doc ──→ refreshStatus() / action result text
TUI: tick ──→ checkHotspotCmd ──→ hotspotStatusMsg ──→ render (no stale Checking...)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/cli/main.go` | Modify | Per-subcommand `FlagSet`s; envelope output; exit `0/2/3/4`; validate URLs/config without panic. |
| `cmd/cli/json.go` | Create | `emitResult`/`emitError`: compact JSON to stdout, diagnostics to stderr. |
| `cmd/conecta/main.go` | Modify | Add `checkHotspot` to tick/refresh keys; real auto-login; reachable message expiry. |
| `cmd/hotspot/main.go` | Modify | Load user config; configured NAT ifaces; fix `CLIENTS (%d)` formatting. |
| `pkg/hotspot/create_ap.go` | Modify | Same-line `Interface <name>` parse; authz pre-check before destructive steps; distinct tool-unavailable errors. |
| `pkg/network/portal.go` | Modify | TLS verify by default; nil-request guard; unknown response → error; no credentials in logs/output. |
| `pkg/network/speed.go` | Modify | Check HTTP status, reject partial non-EOF reads, guard empty URL list; structured result. |
| `omarchy-plugin/bin/*` | Modify | Drop `set -e` early-exit; echo CLI stdout verbatim, propagate exit; surface `speed` result. |
| `omarchy-plugin/Panel.qml` | Modify | Buffer `SplitParser` to a full document before `JSON.parse`; show action/speed results; fix hotspot icon. |
| `omarchy-plugin/manifest.json` | Modify | Align `refreshIntervalSec` key actually read; implement or remove `autoReconnect`. |
| `pkg/config/config.go`, `README.md`, `MANUAL.md`, `omarchy-plugin/README.md` | Modify | `0600` credential files; placeholder credentials; install/privilege/ownership docs. |

## Interfaces / Contracts

```go
type Envelope struct {
  Ok    bool            `json:"ok"`
  Data  json.RawMessage `json:"data,omitempty"`
  Error *ErrBody        `json:"error,omitempty"`
}
type ErrBody struct {
  Code string `json:"code"` // invalid_input|config|unavailable|op_failed|authz
  Msg  string `json:"message"`
}
```

```bash
out=$("$CONNECTA_CLI" "$@" 2>/tmp/c.err); rc=$?
printf '%s' "$out"; exit "$rc"   # never synthesize success; never eval network input
```

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit (`go test`) | `iw` parse, dBm map, envelope writer, login classifier (unknown → not connected) | Table tests on fixture strings; no host tools. |
| Shell | Adapters with stub `CONNECTA_CLI` (exits `0/2/3/4`, hostile SSID) | Assert verbatim passthrough, exit propagation, `jq -e` validity. |
| Host acceptance | Real `iw dev`, `create_ap` path, authz-deny, `SplitParser` fixture, `qmllint` | Manual Omarchy-host checklist; denied authz exits `4`, host untouched. |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|----------|---------------|-----------------|-------------------|
| Documentation-like paths | Applicable: `bin/*` are extensionless executables invoked by path | Always `bash <explicit path>`; never exec-bit/`PATH`-dependent | Decoy same-name binary earlier in `PATH` still resolves correctly |
| Git repository selection | N/A: no git subprocess invoked | — | — |
| Commit state | N/A: no VCS writes | — | — |
| Push state | N/A: no network push automation | — | — |
| PR commands | N/A: no PR/CLI composition | — | — |

Applicable-row rule: fixed argv exec; network/config strings are data only; failures emit failure JSON with the original exit code, host untouched.

## Migration / Rollout

No migration. No daemon, schema, or config-format change; text output is replaced atomically with the JSON contract.

## Open Questions

- [ ] Authorization mechanism for hotspot/NAT/VPN (sudoers drop-in vs polkit vs installer-owned)?
- [ ] Does `SplitParser` aggregate pretty-printed JSON, or is single-line output mandatory? (fixture decides)
- [ ] Approved scope of any TLS exception for the captive portal?
