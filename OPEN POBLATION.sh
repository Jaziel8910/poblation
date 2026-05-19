#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
LAUNCHER="$HOME/.poblation/launcher/bin/poblation-launcher"

echo "POBLATION"
echo "========="
echo

if [ -x "$LAUNCHER" ]; then
  exec "$LAUNCHER"
fi

if command -v go >/dev/null 2>&1 && [ -d "$SCRIPT_DIR/poblation-launcher" ]; then
  cd "$SCRIPT_DIR/poblation-launcher"
  exec go run ./cmd/poblation-launcher
fi

echo "No launcher found."
echo "Windows players should use OPEN POBLATION.bat from the release zip."
echo "On macOS/Linux, clone the repo and run: go run ./poblation-launcher/cmd/poblation-launcher"
exit 1
