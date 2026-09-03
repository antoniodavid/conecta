#!/bin/bash
# Build and deploy conecta: binaries -> $BIN_DIR, plugin -> Omarchy plugins dir.
# Usage: ./deploy.sh [--check] [--no-plugin]
#   --check      run go test ./... before installing (slower, safer)
#   --no-plugin  skip the Omarchy plugin install
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
PLUGIN_DIR="${PLUGIN_DIR:-$HOME/.local/share/omarchy/plugins/conecta}"
CHECK=false
NO_PLUGIN=false

for arg in "$@"; do
  case "$arg" in
    --check) CHECK=true ;;
    --no-plugin) NO_PLUGIN=true ;;
    -h|--help)
      printf '%s\n' \
        'Build and deploy conecta.' \
        'Usage: ./deploy.sh [--check] [--no-plugin]' \
        '  --check      run go test ./... before installing' \
        '  --no-plugin  skip the Omarchy plugin install'
      exit 0
      ;;
    *) echo "Unknown flag: $arg (see --help)" >&2; exit 2 ;;
  esac
done

command -v go >/dev/null || { echo "go not installed" >&2; exit 1; }
cd "$ROOT"

if $CHECK; then
  go test ./...
fi

BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/conecta-build.XXXXXX")"
trap 'rm -rf -- "$BUILD_DIR"' EXIT
mkdir -p "$BIN_DIR"
go build -o "$BUILD_DIR/conecta" ./cmd/conecta/ &
go build -o "$BUILD_DIR/hotspot" ./cmd/hotspot/ &
go build -o "$BUILD_DIR/conecta-cli" ./cmd/cli/ &
wait
install -m755 "$BUILD_DIR/conecta" "$BUILD_DIR/hotspot" "$BUILD_DIR/conecta-cli" "$BIN_DIR/"

if ! $NO_PLUGIN; then
  if [ -d "$(dirname "$PLUGIN_DIR")" ] && [ -w "$(dirname "$PLUGIN_DIR")" ]; then
    mkdir -p "$PLUGIN_DIR"
    cp -a "$ROOT/omarchy-plugin/." "$PLUGIN_DIR/"
  elif command -v sudo >/dev/null; then
    sudo install -d "$PLUGIN_DIR"
    sudo cp -a "$ROOT/omarchy-plugin/." "$PLUGIN_DIR/"
  else
    echo "plugin destination requires write access or sudo: $PLUGIN_DIR" >&2
    exit 1
  fi
  if command -v omarchy >/dev/null; then
    omarchy plugin enable conecta.network || true
  else
    echo "omarchy not found; plugin copied, enable manually" >&2
  fi
fi

if $NO_PLUGIN; then
  echo "deployed binaries to: $BIN_DIR (plugin skipped)"
else
  echo "deployed binaries to: $BIN_DIR + plugin at $PLUGIN_DIR"
fi
