# CLI Plugin Contract Specification

## Purpose

Define a machine-readable boundary between Conecta actions and the Omarchy panel. It MUST NOT introduce a daemon, service, or IPC redesign.

## Requirements

### Requirement: Stable machine-readable responses

Each status/action invocation MUST emit exactly one valid JSON document on stdout; diagnostics MUST remain off it. It MUST identify success/failure and provide result/error. Exit codes MUST be deterministic: `0` success; `2` invalid input/config; `3` unavailable/operation failure; `4` authorization failure.

#### Scenario: Successful action
- GIVEN a valid command and operation
- WHEN the action completes
- THEN stdout contains one parseable success response and exit code is `0`

#### Scenario: Invalid request
- GIVEN an unknown subcommand, flag, or malformed configuration
- WHEN the CLI is invoked
- THEN stdout contains one parseable failure response and exit code is `2`

### Requirement: Correct dispatch and configuration

The CLI MUST bind flags to the subcommand and apply configuration across binaries. Invalid URLs, missing/invalid values, and unsupported options MUST fail without panic or partial action.

#### Scenario: Login arguments and saved settings
- GIVEN credentials supplied through documented flags or configuration
- WHEN `login` is invoked
- THEN those values are used and unrelated subcommands do not consume them

### Requirement: Accurate hotspot and TUI status

Hotspot views MUST report state, clients, and errors, refreshing after startup, stop, or polling. Pending, empty, unavailable, and failed states MUST remain distinguishable; the UI MUST NOT remain at a placeholder indefinitely.

#### Scenario: Status changes after an action
- GIVEN hotspot state changes after start or stop
- WHEN the TUI refreshes
- THEN displayed state and client count reflect the result

### Requirement: Plugin process and JSON failure contract

Plugin actions MUST consume the CLI contract, propagate nonzero exit status, and expose valid JSON on failure. Status handling MUST parse the complete response as one document, including escaped values. A Quickshell fixture MUST verify complete JSON aggregation and the manifest entry point.

#### Scenario: CLI failure from the panel
- GIVEN the CLI exits nonzero
- WHEN a plugin action is run
- THEN the panel receives parseable failure data and does not report success

### Requirement: Configured network status

Status reporting MUST honor configured Wi-Fi/VPN interfaces, preserve signed signal measurements, and distinguish unavailable tools/interfaces from zero signal or no clients. All text MUST remain valid JSON.

#### Scenario: Negative Wi-Fi signal
- GIVEN an interface reports a negative dBm value
- WHEN status is requested
- THEN the signal preserves its sign and maps to the correct strength

### Requirement: Visible speed-test results

A speed-test action MUST expose a parseable result or error to the invoking UI, including completion state and measured outcome. It MUST NOT appear successful while discarding output.

#### Scenario: Speed test result
- GIVEN the endpoint returns valid measurements
- WHEN the speed test runs
- THEN the UI displays the measured result and the CLI returns `0`

### Requirement: Explicit installation and privilege behavior

Installation ownership, required host tools, and non-interactive authorization policy MUST be documented and testable before bar-triggered hotspot, NAT, or VPN actions are enabled. Unauthorized actions MUST fail promptly with their documented status and MUST NOT hang, partially reconfigure networking, or degrade the host. Authorization policy remains an acceptance-tested dependency.

#### Scenario: Authorization is unavailable
- GIVEN authorization is absent or denied
- WHEN a privileged action is requested
- THEN an authorization failure is returned without destructive setup

### Requirement: Safe credentials and TLS

Credentials MUST NOT appear in logs, JSON responses, or examples, and persisted credential configuration MUST be access-restricted. Network clients MUST verify TLS certificates by default; exceptions MUST be explicit, scoped, and authorized. Authentication and malformed responses MUST fail closed.

#### Scenario: Unrecognized portal response
- GIVEN a portal returns an unknown or malformed response
- WHEN login completes
- THEN the result is failure or unknown, never false connected
