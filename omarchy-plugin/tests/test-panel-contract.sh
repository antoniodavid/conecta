#!/bin/bash
# Fixture for the Quickshell BarWidget+Loader contract:
# - manifest entry point resolves to BarWidget.qml (bar slot host)
# - BarWidget owns status polling (refreshIntervalSec, statusScript,
#   statusProcess, refreshStatus) and loads Panel.qml through a Loader
# - Panel is Loader-hosted popup content (no bar button, no poll loop,
#   asks for re-reads via requestRefresh)
# - dead autoReconnect setting is gone from manifest, Panel and BarWidget
# - SplitParser output is buffered to one document before JSON.parse
# - adapters invoked via fixed argv (bash + explicit path)
# - adapters passthrough verbatim with exit propagation (no set -e silence)
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAIL=0

fail() { echo "FAIL: $1"; FAIL=1; }
pass() { echo "PASS: $1"; }

# 1. Manifest is valid JSON and the entry point exists.
ENTRY=$(jq -e -r '.entryPoints.barWidget' "$ROOT/manifest.json" 2>/dev/null) \
  || fail "manifest.json is not valid JSON"
if [ -n "${ENTRY:-}" ] && [ -f "$ROOT/$ENTRY" ]; then
  pass "manifest entry point resolves ($ENTRY)"
else
  fail "manifest entry point missing (got '${ENTRY:-}')"
fi
[ "${ENTRY:-}" = "BarWidget.qml" ] \
  && pass "manifest entry point is BarWidget.qml" \
  || fail "manifest entry point must be BarWidget.qml (got '${ENTRY:-}')"

# 2. refreshIntervalSec key is aligned between manifest and BarWidget (poll owner).
jq -e '.barWidget.defaults.refreshIntervalSec' "$ROOT/manifest.json" >/dev/null 2>&1 \
  && pass "manifest declares refreshIntervalSec" \
  || fail "manifest must declare refreshIntervalSec"
grep -q 'setting("refreshIntervalSec"' "$ROOT/BarWidget.qml" \
  && pass "BarWidget reads refreshIntervalSec" \
  || fail "BarWidget must read setting(\"refreshIntervalSec\")"

# 3. autoReconnect is implemented or removed — it is removed.
if grep -rq 'autoReconnect' "$ROOT/manifest.json" "$ROOT/Panel.qml" "$ROOT/BarWidget.qml"; then
  fail "dead autoReconnect setting still present"
else
  pass "autoReconnect removed from manifest, Panel and BarWidget"
fi

# 4. SplitParser output is buffered to one document before JSON.parse.
# Status side lives in BarWidget, action side in Panel.
for marker in 'statusBuffer' 'feedJson' 'JSON.parse'; do
  grep -q "$marker" "$ROOT/BarWidget.qml" \
    && pass "BarWidget contains $marker" \
    || fail "BarWidget must contain $marker (buffered single-document parse)"
done
for marker in 'actionBuffer' 'feedJson' 'JSON.parse'; do
  grep -q "$marker" "$ROOT/Panel.qml" \
    && pass "Panel contains $marker" \
    || fail "Panel must contain $marker (buffered single-document parse)"
done

# 5. Fixed argv invocation: bash + explicit script path, no PATH lookup.
# Status poll lives in BarWidget, actions in Panel.
grep -q '\[\s*"bash",\s*root\.statusScript' "$ROOT/BarWidget.qml" \
  && pass "BarWidget invokes statusScript via fixed argv" \
  || fail "BarWidget must invoke statusScript as [\"bash\", root.statusScript, ...]"
for script in hotspotScript vpnScript loginScript; do
  grep -q "\[\s*\"bash\",\s*root\.$script" "$ROOT/Panel.qml" \
    && pass "Panel invokes $script via fixed argv" \
    || fail "Panel must invoke $script as [\"bash\", root.$script, ...]"
done

# 6. Adapters: syntax-clean, executable, no set -e early-exit, verbatim passthrough.
for bin in omarchy-conecta-status omarchy-conecta-login omarchy-conecta-hotspot omarchy-conecta-vpn; do
  bash -n "$ROOT/bin/$bin" && pass "$bin syntax" || fail "$bin bash -n"
  [ -x "$ROOT/bin/$bin" ] && pass "$bin executable" || fail "$bin must stay executable"
  if grep -q '^set -.*e' "$ROOT/bin/$bin"; then
    fail "$bin must not use set -e early-exit (swallows failure JSON)"
  else
    pass "$bin no set -e"
  fi
  grep -q 'printf .%s. "\$out"' "$ROOT/bin/$bin" && grep -q 'exit "\$rc"' "$ROOT/bin/$bin" \
    && pass "$bin verbatim passthrough + exit propagation" \
    || fail "$bin must printf verbatim stdout and exit with the CLI code"
done

# 7. Status adapter emits one compact JSON line (SplitParser-safe).
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cat > "$TMP/stub" <<'EOF'
#!/bin/bash
echo '{"ok":true,"data":{"status":"connected","gateway":"g","interface":"i","wifi":{"available":true,"ssid":"s","signal_dbm":-60,"signal":50},"vpn":{"connected":false,"ip":"","interface":"wg0"}}}'
exit 0
EOF
chmod +x "$TMP/stub"
out=$(CONNECTA_CLI="$TMP/stub" bash "$ROOT/bin/omarchy-conecta-status" 2>/dev/null); rc=$?
[ "$rc" -eq 0 ] && pass "status adapter exit 0" || fail "status adapter exit=$rc want 0"
if [ "$(printf '%s' "$out" | wc -l)" -le 1 ] && printf '%s' "$out" | jq -e . >/dev/null 2>&1; then
  pass "status adapter emits one JSON line"
else
  fail "status adapter must emit one JSON line: [$out]"
fi

# 8. VPN profiles + import surface: status forwards profiles, vpn adapter
# supports list/import, Panel lists profiles with fixed-argv connect + import.
grep -q 'profiles' "$ROOT/bin/omarchy-conecta-status" \
  && pass "status adapter forwards vpn profiles" \
  || fail "status adapter must forward vpn profiles"
grep -q 'list|import' "$ROOT/bin/omarchy-conecta-vpn" \
  && pass "vpn adapter supports list/import" \
  || fail "vpn adapter must support list/import actions"
for marker in 'vpnProfiles' 'runVPNConnect' 'Import configs'; do
  grep -q "$marker" "$ROOT/Panel.qml" \
    && pass "Panel contains $marker" \
    || fail "Panel must contain $marker (profile list + import)"
done
grep -q '"connect", name' "$ROOT/Panel.qml" \
  && pass "Panel connects by fixed argv name" \
  || fail "Panel must connect via fixed argv [\"bash\", root.vpnScript, \"connect\", name]"
if grep -F -q '?.' "$ROOT/Panel.qml"; then
  fail "Panel must not use ?. operator"
else
  pass "Panel has no ?. operator"
fi
if grep -F -q '??' "$ROOT/Panel.qml"; then
  fail "Panel must not use ?? operator"
else
  pass "Panel has no ?? operator"
fi
if grep -F -q '?.' "$ROOT/BarWidget.qml"; then
  fail "BarWidget must not use ?. operator"
else
  pass "BarWidget has no ?. operator"
fi
if grep -F -q '??' "$ROOT/BarWidget.qml"; then
  fail "BarWidget must not use ?? operator"
else
  pass "BarWidget has no ?? operator"
fi

# 8b. Connection card shows the logged-in user via the connUser helper.
grep -q 'connUser' "$ROOT/Panel.qml" \
  && pass "Panel contains connUser helper" \
  || fail "Panel must contain connUser helper"
grep -Fq '"User"' "$ROOT/Panel.qml" \
  && pass "Panel contains User row marker" \
  || fail "Panel must contain a \"User\" row label"

# 8c. Panel maps the "no portal" connection status to a neutral pill
# (not a red failure state).
grep -q '"No portal"' "$ROOT/Panel.qml" \
  && pass "Panel maps no portal to a neutral pill" \
  || fail "Panel must map no portal to a neutral pill"

# 9. BarWidget+Loader architecture: BarWidget root, Loader into Panel.qml,
# single IpcHandler on conecta.network with toggle, injectPanel wiring
# anchorItem/hostWidget plus live status/isOn, status poll ownership.
grep -q '^BarWidget' "$ROOT/BarWidget.qml" \
  && pass "BarWidget root is BarWidget" \
  || fail "BarWidget.qml root must be BarWidget (not Panel)"
grep -q 'Panel\.qml' "$ROOT/BarWidget.qml" \
  && pass "BarWidget Loader targets Panel.qml" \
  || fail "BarWidget must load Panel.qml through a Loader"
grep -q 'IpcHandler' "$ROOT/BarWidget.qml" \
  && pass "BarWidget owns IpcHandler" \
  || fail "BarWidget must own the single IpcHandler"
grep -q 'target: "conecta.network"' "$ROOT/BarWidget.qml" \
  && pass "BarWidget IpcHandler targets conecta.network" \
  || fail "BarWidget IpcHandler must target conecta.network"
grep -q 'function toggle()' "$ROOT/BarWidget.qml" \
  && pass "BarWidget IpcHandler exposes toggle" \
  || fail "BarWidget IpcHandler must expose toggle"
grep -q 'function refreshStatus()' "$ROOT/BarWidget.qml" \
  && pass "BarWidget owns refreshStatus" \
  || fail "BarWidget must own refreshStatus (poll owner)"
grep -q 'id: statusProcess' "$ROOT/BarWidget.qml" \
  && pass "BarWidget owns statusProcess" \
  || fail "BarWidget must own statusProcess"
for marker in 'anchorItem' 'hostWidget'; do
  grep -q "$marker" "$ROOT/BarWidget.qml" \
    && pass "BarWidget injectPanel wires $marker" \
    || fail "BarWidget injectPanel must wire $marker"
done
grep -q 'requestRefresh' "$ROOT/BarWidget.qml" \
  && pass "BarWidget hooks panel requestRefresh" \
  || fail "BarWidget must connect panel requestRefresh to refreshStatus"
grep -q 'function refreshSoon()' "$ROOT/BarWidget.qml" \
  && pass "BarWidget owns refreshSoon" \
  || fail "BarWidget must own refreshSoon (immediate + settle refresh)"
grep -q 'requestRefresh.connect(root.refreshSoon)' "$ROOT/BarWidget.qml" \
  && pass "BarWidget routes requestRefresh to refreshSoon" \
  || fail "BarWidget must connect panel requestRefresh to refreshSoon"
for marker in 'settleLeft' 'id: settleTimer'; do
  grep -q "$marker" "$ROOT/BarWidget.qml" \
    && pass "BarWidget contains $marker" \
    || fail "BarWidget must contain $marker (settle follow-ups)"
done
grep -q 'requestRefresh' "$ROOT/Panel.qml" \
  && pass "Panel signals requestRefresh" \
  || fail "Panel must emit requestRefresh instead of polling itself"
if grep -q 'id: statusProcess' "$ROOT/Panel.qml"; then
  fail "Panel must not own statusProcess (moved to BarWidget)"
else
  pass "Panel has no statusProcess"
fi
if grep -q 'BarIconButton' "$ROOT/Panel.qml"; then
  fail "Panel must not own the bar button (moved to BarWidget)"
else
  pass "Panel has no BarIconButton"
fi

if [ "$FAIL" -ne 0 ]; then
  echo "PANEL CONTRACT: FAILING"
  exit 1
fi
echo "PANEL CONTRACT: GREEN"
