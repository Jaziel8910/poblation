#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

if ! command -v go >/dev/null 2>&1; then
  echo "go is not installed or not on PATH." >&2
  exit 1
fi

GOVERSION="$(go env GOVERSION 2>/dev/null || true)"
if [ -z "$GOVERSION" ]; then
  GOVERSION="$(go version | awk '{print $3}')"
fi
GOVERSION="${GOVERSION#go}"

major="$(printf '%s' "$GOVERSION" | awk -F. '{print $1}')"
minor="$(printf '%s' "$GOVERSION" | awk -F. '{print $2}')"

if [ "${major:-0}" -lt 1 ] || { [ "${major:-0}" -eq 1 ] && [ "${minor:-0}" -lt 22 ]; }; then
  echo "Go 1.22 or newer is required. Found $GOVERSION." >&2
  exit 1
fi

cd "$ROOT_DIR"
go install .

mkdir -p "$HOME/.poblation/saves"

cat <<'EOF'
POBLATION is ready.
Your save folder is at ~/.poblation/saves/
Launch the game with the installed binary or run it from this checkout.
EOF
