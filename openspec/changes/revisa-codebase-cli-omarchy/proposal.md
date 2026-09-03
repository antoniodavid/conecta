# Proposal: Make the Conecta CLI and Omarchy Plugin Dependable

## Intent

Make CLI-to-Omarchy interactions dependable by fixing parsing, hotspot/TUI, plugin process/JSON, and privilege defects. Make CLI output/exit status the machine-readable boundary, not duplicated shell logic.

## Scope

### In Scope
- Fix subcommand flags, config loading, Wi-Fi detection, hotspot status polling, and speed-test display.
- Define JSON success/failure responses and exit statuses; make plugin scripts thin adapters that propagate process errors.
- Fix signal/VPN/config handling and JSON escaping; validate Quickshell `SplitParser` with a runtime fixture.
- Document and enforce installation/privilege behavior before bar-triggered network actions.
- Add focused parser and process-failure tests.

### Out of Scope
- A daemon, service, or IPC redesign; defer it until the CLI contract is insufficient.
- Full QML redesign, unrelated refactors, or broad host-network changes.

## Capabilities

### New Capabilities
- `cli-plugin-contract`: Stable machine-readable CLI status/action responses, exit statuses, and plugin consumption.

### Modified Capabilities
- None; no existing main specs are present.

## Approach

Follow the exploration recommendation: repair Go seams, preserve existing binaries/packages, expose one JSON contract, and keep shell/QML thin. Treat privilege authorization and `SplitParser` aggregation as explicit dependencies.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/cli`, `cmd/conecta`, `cmd/hotspot` | Modified | CLI dispatch/config, action output, and TUI. |
| `pkg/hotspot`, `pkg/network` | Modified | Detection and error handling. |
| `omarchy-plugin/{Panel.qml,bin/*,manifest.json}` | Modified | Process, JSON, settings, and actions. |
| `README.md`, `MANUAL.md`, `omarchy-plugin/README.md` | Modified | Installation, privilege, and credentials. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Pre-existing edits overlap affected files | High | Review the diff; isolate owned hunks. |
| GUI privilege authorization is unusable | High | Resolve authorization before enabling actions. |
| `SplitParser` aggregation differs | Med | Add a runtime fixture and compatible fallback. |
| Host services escape unit-test coverage | Med | Separate unit checks from host acceptance checks. |

## Rollback Plan

Revert this change as one unit. No daemon, migration, or schema is introduced; unrelated worktree edits remain untouched.

## Dependencies

- Decision on non-interactive privilege authorization and installation ownership.
- Quickshell fixture confirming `SplitParser` aggregation and manifest entry point.
- Host tools/services (`iw`, `create_ap`, NAT/VPN tooling) for acceptance checks.

## Success Criteria

- [ ] `go test ./...`, `go vet ./...`, `go build ./...`, `bash -n omarchy-plugin/bin/*`, and `qmllint` pass.
- [ ] Documented commands bind flags correctly; every plugin action emits valid JSON and deterministic exit status.
- [ ] Real `iw dev` output, configured interfaces, TUI refresh, and speed-test results work end to end.
- [ ] Privilege/install behavior is documented and authorized; actions do not silently hang or degrade the host.
