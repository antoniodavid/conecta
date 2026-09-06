#!/bin/bash
# RED 3.1 (threat matrix, documentation-like paths):
# - decoy conecta-cli earlier in PATH still resolves via fixed argv (CONNECTA_CLI)
# - stub matrix (0/2/3/4, hostile SSID) asserts verbatim passthrough, exit propagation, jq -e
# Fails before adapters are fixed (they synthesize success + swallow exits).
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin"
FAIL=0

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Stub CLI: behavior driven by STUB_MODE env.
cat > "$TMP/stub-cli" <<'EOF'
#!/bin/bash
mode="${STUB_MODE:-ok}"
case "$mode" in
  ok)      echo '{"ok":true,"data":{"status":"connected"}}'; exit 0 ;;
  invalid)  echo '{"ok":false,"error":{"code":"invalid_input","message":"bad"}}'; exit 2 ;;
  opfail)   echo '{"ok":false,"error":{"code":"op_failed","message":"boom"}}'; exit 3 ;;
  authz)    echo '{"ok":false,"error":{"code":"authz","message":"denied"}}'; exit 4 ;;
  hostile)  echo '{"ok":true,"data":{"ssid":"a\"b\\c","signal":-70}}'; exit 0 ;;
esac
EOF
chmod +x "$TMP/stub-cli"

# Decoy earlier in PATH that must NOT be used when CONNECTA_CLI is fixed.
mkdir -p "$TMP/decoybin"
cat > "$TMP/decoybin/conecta-cli" <<'EOF'
#!/bin/bash
echo '{"ok":true,"data":{"decoy":true}}'
exit 0
EOF
chmod +x "$TMP/decoybin/conecta-cli"

check() {
  local desc="$1"; shift
  if ! "$@" >/dev/null 2>&1; then
    echo "FAIL: $desc"
    FAIL=1
  else
    echo "PASS: $desc"
  fi
}

# 1. Decoy test: with decoy first in PATH but CONNECTA_CLI fixed, output must come from stub.
export PATH="$TMP/decoybin:$PATH"
export CONNECTA_CLI="$TMP/stub-cli"
export STUB_MODE=ok
out=$(bash "$BIN/omarchy-conecta-hotspot" status 2>/dev/null); rc=$?
# hotspot status currently does NOT use CONNECTA_CLI for status (it shells out to systemctl) —
# so we test login/status paths that must go through the CLI verbatim:
out=$(STUB_MODE=hostile bash "$BIN/omarchy-conecta-login" speed 2>/dev/null); rc=$?
if echo "$out" | grep -q '"decoy"'; then
  echo "FAIL: decoy in PATH hijacked adapter (must use fixed CONNECTA_CLI argv)"
  FAIL=1
else
  echo "PASS: decoy in PATH does not hijack adapter"
fi

# 2. Verbatim passthrough + exit propagation + jq validity for login speed (hostile SSID).
export STUB_MODE=hostile
out=$(bash "$BIN/omarchy-conecta-login" speed 2>/dev/null); rc=$?
echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: hostile output is not valid JSON: $out"; FAIL=1; }
echo "$out" | jq -e '.data.ssid == "a\"b\\c"' >/dev/null 2>&1 || { echo "FAIL: hostile SSID not passed verbatim: $out"; FAIL=1; }
[ "$rc" -eq 0 ] || { echo "FAIL: exit 0 not propagated (got $rc)"; FAIL=1; }

# 3. Exit propagation matrix via hotspot start/stop (must propagate stub exits, not synthesize success).
for mode in invalid opfail authz; do
  case "$mode" in
    invalid) want=2;; opfail) want=3;; authz) want=4;;
  esac
  export STUB_MODE="$mode"
  out=$(bash "$BIN/omarchy-conecta-hotspot" start 2>/dev/null); rc=$?
  echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: hotspot start ($mode) not valid JSON: $out"; FAIL=1; }
  if [ "$rc" -ne "$want" ]; then
    echo "FAIL: hotspot start ($mode) exit=$rc want $want (out=$out)"
    FAIL=1
  else
    echo "PASS: hotspot start ($mode) exit $want"
  fi
  # Must not report success on failure.
  if echo "$out" | grep -q '"success":true'; then
    echo "FAIL: hotspot start ($mode) reports success on failure: $out"
    FAIL=1
  fi
done

# 4. VPN passthrough must propagate failures too.
export STUB_MODE=opfail
out=$(bash "$BIN/omarchy-conecta-vpn" connect 2>/dev/null); rc=$?
echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: vpn connect not valid JSON: $out"; FAIL=1; }
if [ "$rc" -ne 3 ]; then
  echo "FAIL: vpn connect exit=$rc want 3"
  FAIL=1
else
  echo "PASS: vpn connect exit 3"
fi

# 5. login action must emit parseable failure JSON on op failure (currently swallows to empty).
export STUB_MODE=opfail
out=$(bash "$BIN/omarchy-conecta-login" login 2>/dev/null); rc=$?
echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: login login ($STUB_MODE) not valid JSON: [$out]"; FAIL=1; }
if [ "$rc" -ne 3 ]; then
  echo "FAIL: login login exit=$rc want 3"
  FAIL=1
else
  echo "PASS: login login exit 3"
fi

# 6. status adapter must emit one valid JSON doc even when CLI fails (no early-exit silence).
export STUB_MODE=opfail
out=$(bash "$BIN/omarchy-conecta-status" 2>/dev/null); rc=$?
echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: status on CLI failure not valid JSON: [$out]"; FAIL=1; }

# 7. VPN list: stub profiles JSON must pass through verbatim with exit 0.
cat > "$TMP/stub-vpn-list" <<'EOF'
#!/bin/bash
if [ "${1:-}" = "vpn" ] && [ "${2:-}" = "list" ]; then
  echo '{"ok":true,"data":{"profiles":[{"name":"USA","active":true,"device":"USA","ip":"10.14.0.2/16","country":"US","flag":"FLAG"}],"count":1}}'
  exit 0
fi
echo '{"ok":false,"error":{"code":"op_failed","message":"unexpected args"}}'
exit 3
EOF
chmod +x "$TMP/stub-vpn-list"
out=$(CONNECTA_CLI="$TMP/stub-vpn-list" bash "$BIN/omarchy-conecta-vpn" list 2>/dev/null); rc=$?
echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: vpn list not valid JSON: [$out]"; FAIL=1; }
echo "$out" | jq -e '.data.profiles[0].name == "USA"' >/dev/null 2>&1 || { echo "FAIL: vpn list profiles not passed verbatim: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: vpn list exit=$rc want 0"
  FAIL=1
else
  echo "PASS: vpn list passthrough exit 0"
fi

# 8. VPN import with missing dir: adapter creates it, none found = ok:true imported:[].
export CONECTA_IMPORT_DIR="$TMP/missing-import-dir"
rm -rf "$CONECTA_IMPORT_DIR"
out=$(CONNECTA_CLI="$TMP/stub-vpn-list" CONECTA_IMPORT_DIR="$CONECTA_IMPORT_DIR" bash "$BIN/omarchy-conecta-vpn" import 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == true and (.data.imported | length == 0)' >/dev/null 2>&1 || { echo "FAIL: vpn import missing dir must be ok:true imported:[]: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: vpn import missing dir exit=$rc want 0"
  FAIL=1
else
  echo "PASS: vpn import missing dir exit 0"
fi
[ -d "$CONECTA_IMPORT_DIR" ] || { echo "FAIL: vpn import must create missing import dir"; FAIL=1; }

# 9. VPN import with existing dir but no confs: same empty success.
mkdir -p "$TMP/empty-import"
out=$(CONNECTA_CLI="$TMP/stub-vpn-list" CONECTA_IMPORT_DIR="$TMP/empty-import" bash "$BIN/omarchy-conecta-vpn" import 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == true and (.data.imported | length == 0)' >/dev/null 2>&1 || { echo "FAIL: vpn import no confs must be ok:true imported:[]: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: vpn import no confs exit=$rc want 0"
  FAIL=1
else
  echo "PASS: vpn import no confs exit 0"
fi

# 10. VPN import mixed success/failure aggregation propagates the CLI failure.
mkdir -p "$TMP/mixed-import"
touch "$TMP/mixed-import/a.conf" "$TMP/mixed-import/b.conf"
cat > "$TMP/stub-vpn-mixed" <<'EOF'
#!/bin/bash
case "${3:-}" in
  *a.conf) echo '{"ok":true,"data":{"action":"import","name":"A","file":"a"}}'; exit 0 ;;
  *b.conf) echo '{"ok":false,"error":{"code":"op_failed","message":"boom b"}}'; exit 3 ;;
  *) echo '{"ok":false,"error":{"code":"op_failed","message":"unexpected"}}'; exit 3 ;;
esac
EOF
chmod +x "$TMP/stub-vpn-mixed"
out=$(CONNECTA_CLI="$TMP/stub-vpn-mixed" CONECTA_IMPORT_DIR="$TMP/mixed-import" bash "$BIN/omarchy-conecta-vpn" import 2>/dev/null); rc=$?
echo "$out" | jq -e . >/dev/null 2>&1 || { echo "FAIL: vpn import mixed not valid JSON: [$out]"; FAIL=1; }
echo "$out" | jq -e '.ok == false and (.data.imported | length == 1) and (.data.errors | length == 1)' >/dev/null 2>&1 || { echo "FAIL: vpn import mixed must aggregate 1 imported + 1 error: [$out]"; FAIL=1; }
if [ "$rc" -ne 3 ]; then
  echo "FAIL: vpn import mixed exit=$rc want 3 (propagate CLI failure, never synthesize success)"
  FAIL=1
else
  echo "PASS: vpn import mixed exit 3"
fi

# 11. VPN import with a file arg passes through the CLI verbatim.
cat > "$TMP/stub-vpn-single" <<'EOF'
#!/bin/bash
if [ "${1:-}" = "vpn" ] && [ "${2:-}" = "import" ]; then
  echo '{"ok":true,"data":{"action":"import","name":"Solo","file":"x"}}'
  exit 0
fi
echo '{"ok":false,"error":{"code":"op_failed","message":"unexpected"}}'
exit 3
EOF
chmod +x "$TMP/stub-vpn-single"
out=$(CONNECTA_CLI="$TMP/stub-vpn-single" bash "$BIN/omarchy-conecta-vpn" import /tmp/solo.conf 2>/dev/null); rc=$?
echo "$out" | jq -e '.data.name == "Solo"' >/dev/null 2>&1 || { echo "FAIL: vpn import single file not passed verbatim: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: vpn import single exit=$rc want 0"
  FAIL=1
else
  echo "PASS: vpn import single exit 0"
fi

# 12. login/logout/speed failure matrix: failure JSON + propagated exit,
# never synthesized success (logout success only on exit 0).
for action in login logout speed; do
  for mode in invalid opfail authz; do
    case "$mode" in
      invalid) want=2;; opfail) want=3;; authz) want=4;;
    esac
    export STUB_MODE="$mode"
    out=$(bash "$BIN/omarchy-conecta-login" "$action" 2>/dev/null); rc=$?
    echo "$out" | jq -e '.ok == false' >/dev/null 2>&1 || { echo "FAIL: login $action ($mode) must be failure JSON: [$out]"; FAIL=1; }
    if [ "$rc" -ne "$want" ]; then
      echo "FAIL: login $action ($mode) exit=$rc want $want"
      FAIL=1
    else
      echo "PASS: login $action ($mode) exit $want"
    fi
    if echo "$out" | grep -q '"success":true'; then
      echo "FAIL: login $action ($mode) reports success on failure: [$out]"
      FAIL=1
    fi
  done
done

# 13. logout success passthrough: stub ok yields verbatim JSON + exit 0.
export STUB_MODE=ok
out=$(bash "$BIN/omarchy-conecta-login" logout 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == true' >/dev/null 2>&1 || { echo "FAIL: login logout ok must be success JSON: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: login logout ok exit=$rc want 0"
  FAIL=1
else
  echo "PASS: login logout ok exit 0"
fi

# 14. CLI failing with empty stdout still yields one failure JSON doc +
# propagated exit (covers crashed/missing CLI: no silence, one JSON line).
cat > "$TMP/stub-empty" <<'EOF'
#!/bin/bash
echo "dial tcp 10.180.0.30:8443: i/o timeout" >&2
exit 3
EOF
chmod +x "$TMP/stub-empty"
out=$(CONNECTA_CLI="$TMP/stub-empty" bash "$BIN/omarchy-conecta-login" logout 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == false' >/dev/null 2>&1 || { echo "FAIL: login logout empty-stdout must be failure JSON: [$out]"; FAIL=1; }
if [ "$rc" -ne 3 ]; then
  echo "FAIL: login logout empty-stdout exit=$rc want 3"
  FAIL=1
else
  echo "PASS: login logout empty-stdout exit 3"
fi

# 15. VPN disconnect [name]: dedicated block forwards the name as fixed
# argv; without a name the generic fallthrough runs. CLI failures propagate
# their failure JSON + exit; success passes through verbatim.
cat > "$TMP/stub-vpn-disc" <<'EOF'
#!/bin/bash
if [ "${1:-}" != "vpn" ] || [ "${2:-}" != "disconnect" ]; then
  echo '{"ok":false,"error":{"code":"op_failed","message":"unexpected"}}'
  exit 3
fi
case "${3:-}" in
  Nope) echo '{"ok":false,"error":{"code":"invalid_input","message":"unknown VPN profile \"Nope\" (available: USA)"}}'; exit 2 ;;
  *) echo '{"ok":true,"data":{"action":"disconnect","connected":false,"name":"USA"}}'; exit 0 ;;
esac
EOF
chmod +x "$TMP/stub-vpn-disc"

# 15a. disconnect with a name, success: verbatim passthrough + exit 0.
out=$(CONNECTA_CLI="$TMP/stub-vpn-disc" bash "$BIN/omarchy-conecta-vpn" disconnect USA 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == true and .data.name == "USA"' >/dev/null 2>&1 || { echo "FAIL: vpn disconnect USA not passed verbatim: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: vpn disconnect USA exit=$rc want 0"
  FAIL=1
else
  echo "PASS: vpn disconnect USA passthrough exit 0"
fi

# 15b. disconnect without a name, success: generic fallthrough, verbatim + exit 0.
out=$(CONNECTA_CLI="$TMP/stub-vpn-disc" bash "$BIN/omarchy-conecta-vpn" disconnect 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == true and .data.name == "USA"' >/dev/null 2>&1 || { echo "FAIL: vpn disconnect no-name not passed verbatim: [$out]"; FAIL=1; }
if [ "$rc" -ne 0 ]; then
  echo "FAIL: vpn disconnect no-name exit=$rc want 0"
  FAIL=1
else
  echo "PASS: vpn disconnect no-name passthrough exit 0"
fi

# 15c. disconnect with an unknown name: failure JSON + propagated exit 2.
out=$(CONNECTA_CLI="$TMP/stub-vpn-disc" bash "$BIN/omarchy-conecta-vpn" disconnect Nope 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == false and .error.code == "invalid_input"' >/dev/null 2>&1 || { echo "FAIL: vpn disconnect Nope must be invalid_input failure JSON: [$out]"; FAIL=1; }
if [ "$rc" -ne 2 ]; then
  echo "FAIL: vpn disconnect Nope exit=$rc want 2"
  FAIL=1
else
  echo "PASS: vpn disconnect Nope exit 2"
fi

# 15d. disconnect without a name, CLI failure: failure JSON + propagated exit 3.
export STUB_MODE=opfail
out=$(bash "$BIN/omarchy-conecta-vpn" disconnect 2>/dev/null); rc=$?
echo "$out" | jq -e '.ok == false and .error.code == "op_failed"' >/dev/null 2>&1 || { echo "FAIL: vpn disconnect opfail must be failure JSON: [$out]"; FAIL=1; }
if [ "$rc" -ne 3 ]; then
  echo "FAIL: vpn disconnect opfail exit=$rc want 3"
  FAIL=1
else
  echo "PASS: vpn disconnect opfail exit 3"
fi

if [ "$FAIL" -ne 0 ]; then
  echo "ADAPTER CONTRACT: RED (failing as expected before fix)"
  exit 1
fi
echo "ADAPTER CONTRACT: GREEN"
