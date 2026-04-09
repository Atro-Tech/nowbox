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
  # Resolve version from GitHub API
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//' || echo "latest")
  echo "nowbox: downloading ${VERSION}..." >&2
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

    # Download native (CGO) build for the .app bundle
    APP_NAME="nowbox-${OS}-${ARCH}-app"
    echo "nowbox: downloading native build..." >&2
    APP_TMP="$CACHE_DIR/.nowbox-app-$$"
    $FETCH "${BASE_URL}/${APP_NAME}" > "$APP_TMP" 2>/dev/null || {
      echo "nowbox: native build not available, using standard binary" >&2
      APP_TMP="$TMP"
    }
    chmod +x "$APP_TMP"

    # Copy native binary into .app and ad-hoc sign
    cp "$APP_TMP" "$APP_DIR/Contents/MacOS/nowbox-bin"
    chmod +x "$APP_DIR/Contents/MacOS/nowbox-bin"
    codesign --sign - --force "$APP_DIR/Contents/MacOS/nowbox-bin" 2>/dev/null
    [ "$APP_TMP" != "$TMP" ] && rm -f "$APP_TMP"

    # Compile native launcher — handles Finder "Open With" via Apple Events
    cat > "$CACHE_DIR/launcher.swift" << 'SWIFT'
import Cocoa

class Delegate: NSObject, NSApplicationDelegate {
    func application(_ app: NSApplication, open urls: [URL]) {
        let bin = Bundle.main.url(forAuxiliaryExecutable: "nowbox-bin")!.path
        for url in urls where url.pathExtension == "now" {
            guard let content = try? String(contentsOf: url, encoding: .utf8) else { continue }
            var token = ""
            if let r = content.range(of: "NOWBOX_TOKEN=\""),
               let end = content[r.upperBound...].firstIndex(of: "\"") {
                token = String(content[r.upperBound..<end])
            }
            let task = Process()
            task.executableURL = URL(fileURLWithPath: bin)
            task.arguments = ["-client", "app"]
            task.environment = ProcessInfo.processInfo.environment
            task.environment?["NOWBOX_TOKEN"] = token
            task.environment?["NOWBOX_FILE"] = url.path
            try? task.run()
        }
    }
    func applicationDidFinishLaunching(_ n: Notification) {
        // Quit if launched without files (e.g. clicking dock icon)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) {
            NSApp.terminate(nil)
        }
    }
}
let delegate = Delegate()
let app = NSApplication.shared
app.delegate = delegate
app.run()
SWIFT
    swiftc -O -o "$APP_DIR/Contents/MacOS/nowbox" "$CACHE_DIR/launcher.swift" 2>/dev/null
    rm -f "$CACHE_DIR/launcher.swift"

    # Also keep a shell entry point for CLI use
    cat > "$APP_DIR/Contents/MacOS/nowbox-cli" << 'SHIM'
#!/bin/sh
exec "$(dirname "$0")/nowbox-bin" "$@"
SHIM
    chmod +x "$APP_DIR/Contents/MacOS/nowbox-cli"

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
      <key>LSHandlerRank</key>
      <string>Owner</string>
      <key>LSItemContentTypes</key>
      <array>
        <string>lol.nowbox.session</string>
      </array>
    </dict>
  </array>
  <key>UTExportedTypeDeclarations</key>
  <array>
    <dict>
      <key>UTTypeIdentifier</key>
      <string>lol.nowbox.session</string>
      <key>UTTypeDescription</key>
      <string>nowbox session</string>
      <key>UTTypeConformsTo</key>
      <array>
        <string>public.shell-script</string>
      </array>
      <key>UTTypeTagSpecification</key>
      <dict>
        <key>public.filename-extension</key>
        <array>
          <string>now</string>
        </array>
      </dict>
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

    # Sign the whole bundle and register file association
    codesign --sign - --force --deep "$APP_DIR" 2>/dev/null
    /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$APP_DIR" 2>/dev/null || true
    # Set nowbox as default for .now files using Swift (reliable on modern macOS)
    swift -e '
import Foundation
import UniformTypeIdentifiers
if #available(macOS 12.0, *) {
  let ws = NSWorkspace.shared
  let url = URL(fileURLWithPath: "/Applications/nowbox.app")
  if let ut = UTType("lol.nowbox.session") {
    ws.setDefaultApplication(at: url, toOpenContentType: ut) { error in
      if let e = error { fputs("warn: \(e)\n", stderr) }
    }
  }
}
' 2>/dev/null || true

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

# ── Normal mode: always download fresh ──
rm -rf "$CACHE_DIR"
rm -rf "${HOME}/.nowbox"

TMP=$(download_binary)
mv "$TMP" "$BINARY"

echo "nowbox: ready" >&2
exec "$BINARY" "$@" </dev/tty
