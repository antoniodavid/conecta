#!/bin/bash
# Fixture for the Quickshell panel contract (task 3.4):
# - manifest entry point resolves to a real file
# - Panel reads the same refreshIntervalSec key the manifest declares
# - dead autoReconnect setting is gone from manifest and Panel
# - Panel buffers SplitParser chunks into one document before JSON.parse
# - Panel invokes adapters via fixed argv (bash + explicit path)
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

# 2. refreshIntervalSec key is aligned between manifest and Panel.
jq -e '.barWidget.defaults.refreshIntervalSec' "$ROOT/manifest.json" >/dev/null 2>&1 \
  && pass "manifest declares refreshIntervalSec" \
  || fail "manifest must declare refreshIntervalSec"
grep -q 'setting("refreshIntervalSec"' "$ROOT/Panel.qml" \
  && pass "Panel reads refreshIntervalSec" \
  || fail "Panel must read setting(\"refreshIntervalSec\")"

# 3. autoReconnect is implemented or removed — it is removed.
if grep -rq 'autoReconnect' "$ROOT/manifest.json" "$ROOT/Panel.qml"; then
  fail "dead autoReconnect setting still present"
else
  pass "autoReconnect removed from manifest and Panel"
fi

# 4. SplitParser output is buffered to one document before JSON.parse.
for marker in 'statusBuffer' 'actionBuffer' 'feedJson' 'JSON.parse'; do
  grep -q "$marker" "$ROOT/Panel.qml" \
    && pass "Panel contains $marker" \
    || fail "Panel must contain $marker (buffered single-document parse)"
done

# 5. Fixed argv invocation: bash + explicit script path, no PATH lookup.
for script in statusScript hotspotScript vpnScript loginScript; do
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

if [ "$FAIL" -ne 0 ]; then
  echo "PANEL CONTRACT: FAILING"
  exit 1
fi
echo "PANEL CONTRACT: GREEN"
