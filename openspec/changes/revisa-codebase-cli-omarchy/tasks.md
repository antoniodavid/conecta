# Tasks: Make the Conecta CLI and Omarchy Plugin Dependable

## Review Workload Forecast

- Estimated changed lines: 650–750 (additions + deletions)
- Delivery strategy: single-pr; split Units 1 → 2 → 3 (or single PR under size:exception)

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

Single-pr requires size:exception; fits the 800-line budget.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | CLI contract (envelope, FlagSet, validation) | slice 1 | `go test ./cmd/cli/...` | `conecta-cli <cmd> \| jq -e .` + exit check | `cmd/cli/*` |
| 2 | Core safety (detect, privilege, TLS, speed, TUI) | slice 2 | `go test ./pkg/... ./cmd/...` | TUI refresh + `iw dev` fixture + speed run | `pkg/*`, `cmd/conecta`, `cmd/hotspot` |
| 3 | Plugin + hardening (adapters, QML, config, docs) | slice 3 | `bash -n omarchy-plugin/bin/* && qmllint omarchy-plugin/Panel.qml` | Host checklist (Phase 4) | `omarchy-plugin/*`, `pkg/config`, docs |

## Phase 1: Contract foundation

- [x] 1.1 RED: failing `cmd/cli/json.go` envelope tests (one JSON line on stdout, stderr diagnostics, exits 0/2/3/4).
- [x] 1.2 GREEN: create `cmd/cli/json.go` (`emitResult`/`emitError`, `Envelope`/`ErrBody`).
- [x] 1.3 `cmd/cli/main.go`: per-subcommand `FlagSet`s; `login --user/--pass` binds to login only; validate URLs/config without panic or partial action.
- [x] 1.4 `cmd/cli/main.go`: envelope + exits on every path (bad input/config → 2, op failure → 3).

## Phase 2: Core implementation

- [x] 2.1 RED: tests for `Interface <name>` parse, signed dBm, login classifier (unknown → failure), speed rejection (bad status/partial read/empty URLs).
- [x] 2.2 `pkg/hotspot/create_ap.go`: same-line detection; authz pre-check before destructive steps (fail closed, exit 4); distinct tool-unavailable errors.
- [x] 2.3 `pkg/network/portal.go`: TLS verify default, nil-request guard, no credential leaks; `pkg/network/speed.go`: status check, EOF-only success, empty-list guard, structured result.
- [x] 2.4 `cmd/conecta/main.go`: `checkHotspotCmd` in tick/refresh, real auto-login, message expiry; `cmd/hotspot/main.go`: load user config, configured NAT ifaces, `CLIENTS` fix.

## Phase 3: Plugin integration

- [x] 3.1 RED (threat matrix, documentation-like paths): decoy `conecta-cli` in `PATH` resolves via fixed argv; stub matrix (0/2/3/4, hostile SSID) asserts verbatim passthrough, exit propagation, `jq -e`.
- [x] 3.2 `omarchy-plugin/bin/*`: drop `set -e` early-exit; verbatim passthrough with exit propagation; surface `speed`; configured VPN iface; JSON-escape fields.
- [x] 3.3 `omarchy-plugin/Panel.qml`: buffer `SplitParser` to one document before `JSON.parse`; show action/speed results; fix hotspot icon.
- [x] 3.4 `omarchy-plugin/manifest.json`: align `refreshIntervalSec`; implement or remove `autoReconnect`; add Quickshell fixture (aggregation + entry point).

## Phase 4: Testing and acceptance

- [x] 4.1 `go test ./...`, `go vet ./...`, `go build ./...`; spec scenarios (success → 0, invalid → 2, parseable failure JSON, dBm sign kept, authz-deny → 4).
- [ ] 4.2 Host checklist (manual, Omarchy host): real `iw dev` detected; start/stop reflected in TUI/panel; denied authz exits 4, host unchanged; fixture passes; `qmllint` clean; speed displays. (Not runnable in this container; left for maintainer on Omarchy hardware.)

## Phase 5: Hardening and docs

- [x] 5.1 `pkg/config/config.go`: `0600` credential saves; `MANUAL.md`: placeholder credentials.
- [x] 5.2 `README.md`, `MANUAL.md`, `omarchy-plugin/README.md`: install ownership, host tools, non-interactive authz policy; gate bar-triggered hotspot/NAT/VPN on authorized setup.
