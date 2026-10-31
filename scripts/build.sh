#!/usr/bin/env bash
# Builds Mine Party 95 for every platform into dist/.
set -euo pipefail
cd "$(dirname "$0")/.."

out=dist
rm -rf "$out"
mkdir -p "$out"

echo "linux/amd64"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" \
    -o "$out/mineparty-linux-amd64" ./cmd/mineparty

echo "windows/amd64"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H=windowsgui" \
    -o "$out/mineparty-windows-amd64.exe" ./cmd/mineparty

# macOS gets a .app bundle, so it opens like any other Mac app.
for arch in arm64 amd64; do
    echo "darwin/$arch"
    app="$out/MineParty95-macos-$arch.app"
    mkdir -p "$app/Contents/MacOS"
    GOOS=darwin GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" \
        -o "$app/Contents/MacOS/mineparty" ./cmd/mineparty
    cat > "$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>Mine Party 95</string>
    <key>CFBundleIdentifier</key><string>app.mineparty95</string>
    <key>CFBundleExecutable</key><string>mineparty</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleShortVersionString</key><string>1.0</string>
    <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST
    (cd "$out" && zip -qr "MineParty95-macos-$arch.zip" "MineParty95-macos-$arch.app")
    rm -rf "$app"
done

ls -lh "$out"
