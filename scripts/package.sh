#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
app="$PWD/build/Voice Pill.app"
version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$app/Contents/Info.plist")
mkdir -p dist
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
ditto "$app" "$stage/Voice Pill.app"
ln -s /Applications "$stage/Applications"
cp docs/INSTALL.txt "$stage/READ ME FIRST.txt"
hdiutil create -volname 'Voice Pill' -srcfolder "$stage" -ov -format UDZO "dist/Voice-Pill-${version}-arm64.dmg" >/dev/null
ditto -c -k --sequesterRsrc --keepParent "$app" "dist/Voice-Pill-${version}-arm64.zip"
(cd dist && shasum -a 256 "Voice-Pill-${version}-arm64.dmg" "Voice-Pill-${version}-arm64.zip" > SHA256SUMS)
printf 'Packages: %s/dist\n' "$PWD"
