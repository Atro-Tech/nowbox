#!/bin/sh
set -e

REPO="Atro-Tech/nowbox"
CACHE_DIR="${NOWBOX_CACHE_DIR:-$HOME/.cache/nowbox}"
INSTALL_DIR="${NOWBOX_INSTALL_DIR:-/usr/local/bin}"

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

# Install mode: curl nowbox.lol | sh -s -- install
if [ "$1" = "install" ]; then
  echo "nowbox: installing..." >&2

  TMP="$CACHE_DIR/.nowbox-download-$$"
  mkdir -p "$CACHE_DIR"
  $FETCH "${BASE_URL}/${NAME}" > "$TMP" 2>/dev/null || {
    echo "nowbox: download failed" >&2
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
    fi
    if [ "$ACTUAL" != "$EXPECTED" ]; then
      echo "nowbox: checksum mismatch" >&2
      rm -f "$TMP"
      exit 1
    fi
  fi

  chmod +x "$TMP"

  # Try /usr/local/bin first, fall back to ~/.local/bin
  if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP" "$INSTALL_DIR/nowbox${EXT}"
    echo "nowbox: installed to $INSTALL_DIR/nowbox" >&2
  elif [ -w "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
    mv "$TMP" "$HOME/.local/bin/nowbox${EXT}"
    echo "nowbox: installed to ~/.local/bin/nowbox" >&2
    case "$PATH" in
      *"$HOME/.local/bin"*) ;;
      *) echo "nowbox: add ~/.local/bin to your PATH" >&2 ;;
    esac
  else
    echo "nowbox: cannot write to $INSTALL_DIR or ~/.local/bin" >&2
    echo "nowbox: try: sudo curl -fsSL nowbox.lol | sh -s -- install" >&2
    rm -f "$TMP"
    exit 1
  fi

  # Also cache it
  cp "$INSTALL_DIR/nowbox${EXT}" "$BINARY" 2>/dev/null || \
    cp "$HOME/.local/bin/nowbox${EXT}" "$BINARY" 2>/dev/null || true

  echo "nowbox: done. run: nowbox" >&2
  exit 0
fi

# Normal mode: download to cache and run
if [ -x "$BINARY" ]; then
  exec "$BINARY" "$@" </dev/tty
fi

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
    ACTUAL="$EXPECTED"
  fi
  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "nowbox: checksum mismatch" >&2
    rm -f "$TMP"
    exit 1
  fi
  echo "nowbox: verified" >&2
fi

chmod +x "$TMP"
mv "$TMP" "$BINARY"

echo "nowbox: ready" >&2
exec "$BINARY" "$@" </dev/tty
