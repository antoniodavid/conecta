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
trap - EXIT

echo "installed $TARGET (mode 0440, owned by root)"
echo "granted NOPASSWD root to $USER_NAME for:"
echo "  systemctl is-active|start|stop create_ap"
echo "  tee /etc/create_ap.conf"
echo "  killall hostapd dnsmasq"
echo "  nmcli r wifi off|on"
echo "  rfkill unblock wlan"
echo "hotspot and NAT actions via conecta-cli now work without a password."