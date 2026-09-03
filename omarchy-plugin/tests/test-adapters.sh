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

if [ "$FAIL" -ne 0 ]; then
  echo "ADAPTER CONTRACT: RED (failing as expected before fix)"
  exit 1
fi
echo "ADAPTER CONTRACT: GREEN"
