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

NAME="nowbox-${OS}-${ARCH}${EXT}"
BINARY="$CACHE_DIR/nowbox${EXT}"
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

download_binary() {
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
  echo "$TMP"
}

# ── Install mode ──
if [ "$1" = "install" ]; then
  echo "nowbox: installing..." >&2
  TMP=$(download_binary)

  if [ "$OS" = "darwin" ]; then
    # macOS: create .app bundle in /Applications
    APP_DIR="/Applications/nowbox.app"
    mkdir -p "$APP_DIR/Contents/MacOS"
    mkdir -p "$APP_DIR/Contents/Resources"

    # Copy binary
    cp "$TMP" "$APP_DIR/Contents/MacOS/nowbox-bin"
    chmod +x "$APP_DIR/Contents/MacOS/nowbox-bin"

    # Launcher script — opens Terminal with nowbox
    cat > "$APP_DIR/Contents/MacOS/nowbox" << 'LAUNCHER'
#!/bin/sh
DIR="$(dirname "$0")"
if [ -t 1 ]; then
  exec "$DIR/nowbox-bin" "$@"
else
  osascript -e "tell application \"Terminal\" to do script \"'$DIR/nowbox-bin'\""
  osascript -e "tell application \"Terminal\" to activate"
fi
LAUNCHER
    chmod +x "$APP_DIR/Contents/MacOS/nowbox"

    # Info.plist
    cat > "$APP_DIR/Contents/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>nowbox</string>
  <key>CFBundleIdentifier</key>
  <string>lol.nowbox.app</string>
  <key>CFBundleName</key>
  <string>nowbox</string>
  <key>CFBundleDisplayName</key>
  <string>nowbox</string>
  <key>CFBundleVersion</key>
  <string>0.1.0</string>
  <key>CFBundleShortVersionString</key>
  <string>0.1.0</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleIconFile</key>
  <string>icon</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeExtensions</key>
      <array>
        <string>now</string>
      </array>
      <key>CFBundleTypeName</key>
      <string>nowbox session</string>
      <key>CFBundleTypeRole</key>
      <string>Editor</string>
    </dict>
  </array>
</dict>
</plist>
PLIST

    # Generate icon from SVG using sips (macOS built-in)
    # Create a simple 512x512 PNG icon
    cat > "$CACHE_DIR/icon.svg" << 'ICONSVG'
<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
<rect width="24" height="24" rx="5" fill="#1a1a1a"/>
<g transform="translate(3,3) scale(0.75)">
<path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/>
<path d="m3.3 7 8.7 5 8.7-5"/>
<path d="M12 22V12"/>
</g>
</svg>
ICONSVG

    # Convert SVG to ICNS if possible
    if command -v rsvg-convert >/dev/null 2>&1; then
      rsvg-convert -w 512 -h 512 "$CACHE_DIR/icon.svg" -o "$CACHE_DIR/icon.png"
    elif command -v qlmanage >/dev/null 2>&1; then
      # Use macOS Quick Look to render
      qlmanage -t -s 512 -o "$CACHE_DIR" "$CACHE_DIR/icon.svg" 2>/dev/null || true
      [ -f "$CACHE_DIR/icon.svg.png" ] && mv "$CACHE_DIR/icon.svg.png" "$CACHE_DIR/icon.png"
    fi

    if [ -f "$CACHE_DIR/icon.png" ]; then
      # Create iconset and convert to icns
      ICONSET="$CACHE_DIR/icon.iconset"
      mkdir -p "$ICONSET"
      sips -z 16 16 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_16x16.png" 2>/dev/null
      sips -z 32 32 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_16x16@2x.png" 2>/dev/null
      sips -z 32 32 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_32x32.png" 2>/dev/null
      sips -z 64 64 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_32x32@2x.png" 2>/dev/null
      sips -z 128 128 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_128x128.png" 2>/dev/null
      sips -z 256 256 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_128x128@2x.png" 2>/dev/null
      sips -z 256 256 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_256x256.png" 2>/dev/null
      sips -z 512 512 "$CACHE_DIR/icon.png" --out "$ICONSET/icon_256x256@2x.png" 2>/dev/null
      cp "$CACHE_DIR/icon.png" "$ICONSET/icon_512x512.png"
      iconutil -c icns -o "$APP_DIR/Contents/Resources/icon.icns" "$ICONSET" 2>/dev/null
      rm -rf "$ICONSET" "$CACHE_DIR/icon.png" "$CACHE_DIR/icon.svg"
    fi

    # Also install CLI to PATH
    if [ -w "$INSTALL_DIR" ]; then
      cp "$TMP" "$INSTALL_DIR/nowbox"
    elif mkdir -p "$HOME/.local/bin" 2>/dev/null; then
      cp "$TMP" "$HOME/.local/bin/nowbox"
    fi

    rm -f "$TMP"

    # Register .now file association
    /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$APP_DIR" 2>/dev/null || true

    echo "nowbox: installed to /Applications/nowbox.app" >&2
    echo "nowbox: .now files will open with nowbox" >&2
    echo "nowbox: run: nowbox" >&2

  else
    # Linux/other: install to PATH
    if [ -w "$INSTALL_DIR" ]; then
      mv "$TMP" "$INSTALL_DIR/nowbox${EXT}"
      echo "nowbox: installed to $INSTALL_DIR/nowbox" >&2
    elif mkdir -p "$HOME/.local/bin" 2>/dev/null; then
      mv "$TMP" "$HOME/.local/bin/nowbox${EXT}"
      echo "nowbox: installed to ~/.local/bin/nowbox" >&2
      case "$PATH" in
        *"$HOME/.local/bin"*) ;;
        *) echo "nowbox: add ~/.local/bin to your PATH" >&2 ;;
      esac
    else
      echo "nowbox: cannot install — try with sudo" >&2
      rm -f "$TMP"
      exit 1
    fi
  fi

  echo "nowbox: done" >&2
  exit 0
fi

# ── Normal mode: download to cache and run ──
if [ -x "$BINARY" ]; then
  exec "$BINARY" "$@" </dev/tty
fi

TMP=$(download_binary)
mv "$TMP" "$BINARY"

echo "nowbox: ready" >&2
exec "$BINARY" "$@" </dev/tty
