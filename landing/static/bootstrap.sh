#!/bin/sh
set -e

REPO="nowbox/nowbox"
CACHE_DIR="${NOWBOX_CACHE_DIR:-$HOME/.cache/nowbox}"
BINARY="$CACHE_DIR/nowbox"

# Platform detection
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac

EXT=""
if [ "$OS" = "windows" ] || [ "$OS" = "mingw"* ] || [ "$OS" = "msys"* ]; then
  OS="windows"
  EXT=".exe"
fi

NAME="nowbox-${OS}-${ARCH}${EXT}"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

# Download tool
if command -v curl >/dev/null 2>&1; then
  FETCH="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  FETCH="wget -qO-"
else
  echo "nowbox: error: curl or wget required" >&2
  exit 1
fi

# Check cache
if [ -x "$BINARY" ]; then
  exec "$BINARY" "$@"
fi

# Download
echo "nowbox: downloading..." >&2
mkdir -p "$CACHE_DIR"

TMP="$CACHE_DIR/.nowbox-download-$$"
$FETCH "${BASE_URL}/${NAME}" > "$TMP" 2>/dev/null || {
  echo "nowbox: download failed" >&2
  echo "nowbox: try: ${BASE_URL}/${NAME}" >&2
  rm -f "$TMP"
  exit 1
}

chmod +x "$TMP"
mv "$TMP" "$BINARY"

echo "nowbox: ready" >&2
exec "$BINARY" "$@"
