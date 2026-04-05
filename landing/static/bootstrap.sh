#!/bin/sh
set -e

REPO="Atro-Tech/nowbox"
CACHE_DIR="${NOWBOX_CACHE_DIR:-$HOME/.cache/nowbox}"

# Platform detection
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac

EXT=""
case "$OS" in
  mingw*|msys*|cygwin*|windows*)
    OS="windows"
    EXT=".exe"
    ;;
esac

BINARY="$CACHE_DIR/nowbox${EXT}"

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

# Download binary + checksum
echo "nowbox: downloading..." >&2
mkdir -p "$CACHE_DIR"

TMP="$CACHE_DIR/.nowbox-download-$$"
$FETCH "${BASE_URL}/${NAME}" > "$TMP" 2>/dev/null || {
  echo "nowbox: download failed" >&2
  echo "nowbox: try: ${BASE_URL}/${NAME}" >&2
  rm -f "$TMP"
  exit 1
}

# Verify checksum
EXPECTED=$($FETCH "${BASE_URL}/${NAME}.sha256" 2>/dev/null | awk '{print $1}')
if [ -n "$EXPECTED" ]; then
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
  else
    echo "nowbox: warning: cannot verify checksum (no sha256sum or shasum)" >&2
    ACTUAL="$EXPECTED"
  fi

  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "nowbox: checksum mismatch — download may be corrupted or tampered" >&2
    echo "nowbox: expected: $EXPECTED" >&2
    echo "nowbox: got:      $ACTUAL" >&2
    rm -f "$TMP"
    exit 1
  fi
  echo "nowbox: verified" >&2
else
  echo "nowbox: warning: no checksum available, skipping verification" >&2
fi

chmod +x "$TMP"
mv "$TMP" "$BINARY"

echo "nowbox: ready" >&2
exec "$BINARY" "$@"
