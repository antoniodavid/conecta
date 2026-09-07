#!/bin/bash
# Install the conecta sudoers drop-in: NOPASSWD root for the exact hotspot
# commands the CLI runs (see contrib/sudoers-conecta for the grant list).
# Run once as a user with full sudo: ./deploy.sh --setup-privileges
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$ROOT/contrib/sudoers-conecta"
TARGET="/etc/sudoers.d/conecta"
USER_NAME="$(id -un)"

[ -f "$TEMPLATE" ] || { echo "error: template not found: $TEMPLATE" >&2; exit 1; }

TMP="$(mktemp)"
trap 'rm -f -- "$TMP"' EXIT
sed "s/__USER__/$USER_NAME/g" "$TEMPLATE" > "$TMP"

sudo install -o root -g root -m 0440 "$TMP" "$TARGET" \
  || { echo "error: failed to install $TARGET (is sudo available?)" >&2; exit 1; }
sudo visudo -c -f "$TARGET" \
  || { echo "error: sudoers validation failed for $TARGET; remove it and fix the template" >&2; exit 1; }
rm -f -- "$TMP"

# Install the corrected create_ap systemd unit (ExecStop uses pkill -x so it
# never self-interrupts the stop or skips the radio restore).
UNIT_SRC="$ROOT/contrib/create_ap.service"
UNIT_DST="/etc/systemd/system/create_ap.service"
[ -f "$UNIT_SRC" ] || { echo "error: unit template not found: $UNIT_SRC" >&2; exit 1; }
sudo install -o root -g root -m 0644 "$UNIT_SRC" "$UNIT_DST" \
  || { echo "error: failed to install $UNIT_DST" >&2; exit 1; }
sudo systemctl daemon-reload \
  || { echo "error: systemctl daemon-reload failed" >&2; exit 1; }
trap - EXIT

echo "installed $TARGET (mode 0440, owned by root)"
echo "granted NOPASSWD root to $USER_NAME for:"
echo "  systemctl is-active|start|stop create_ap"
echo "  tee /etc/create_ap.conf"
echo "  killall hostapd dnsmasq"
echo "  nmcli r wifi off|on"
echo "  rfkill unblock wlan"
echo "installed corrected $UNIT_DST (pkill -x, radio restored on stop)"
echo "hotspot and NAT actions via conecta-cli now work without a password."