#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
app="$PWD/build/Voice Pill.app"
version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$app/Contents/Info.plist")
python="${VOICEPILL_DMG_PYTHON:-$PWD/build/dmg-tools/bin/python}"
preview=false
if [ "${1:-}" = '--preview' ]; then preview=true; fi
if ! "$preview"; then
  xcrun stapler validate "$app"
  spctl --assess --type execute --verbose=4 "$app"
fi
if [ ! -x "$python" ]; then
  echo 'Set VOICEPILL_DMG_PYTHON to a Python environment with ds-store==1.3.3 installed.' >&2; exit 1
fi
out="$PWD/dist"
if "$preview"; then out="$PWD/build/installer-preview"; fi
mkdir -p "$out"
work=$(mktemp -d "$PWD/build/dmg.XXXXXX")
mount="$work/mounted"
stage="$work/stage"
mounted=false
cleanup() { if "$mounted"; then hdiutil detach "$mount" >/dev/null || true; fi; rm -rf "$work"; }
trap cleanup EXIT
mkdir -p "$stage/.background" "$mount"
ditto "$app" "$stage/Voice Pill.app"
ln -s /Applications "$stage/Applications"
cp Resources/Installer/background.png "$stage/.background/background.png"
hdiutil create -volname 'Voice Pill' -srcfolder "$stage" -ov -format UDRW "$work/layout.dmg" >/dev/null
hdiutil attach "$work/layout.dmg" -nobrowse -noautoopen -mountpoint "$mount" >/dev/null
mounted=true
"$python" scripts/layout-dmg.py "$mount"
sync
hdiutil detach "$mount" >/dev/null
mounted=false
if "$preview"; then
  hdiutil convert "$work/layout.dmg" -format UDZO -ov -o "$out/Voice-Pill-${version}-arm64.dmg" >/dev/null
else
  hdiutil convert "$work/layout.dmg" -format UDZO -o "$out/Voice-Pill-${version}-arm64.dmg" >/dev/null
fi
if ! "$preview"; then
  if [ -n "${VOICEPILL_DEVELOPER_ID:-}" ]; then
  args=(--force --sign "$VOICEPILL_DEVELOPER_ID" --timestamp)
  if [ -n "${VOICEPILL_SIGNING_KEYCHAIN:-}" ]; then args+=(--keychain "$VOICEPILL_SIGNING_KEYCHAIN"); fi
  codesign "${args[@]}" "$out/Voice-Pill-${version}-arm64.dmg"
  else
    echo "Packaging the stapled Developer ID app; DMG container is unsigned (cloud signing)."
  fi
fi
ditto -c -k --sequesterRsrc --keepParent "$app" "$out/Voice-Pill-${version}-arm64.zip"
hdiutil verify "$out/Voice-Pill-${version}-arm64.dmg" >/dev/null
(cd "$out" && shasum -a 256 "Voice-Pill-${version}-arm64.dmg" "Voice-Pill-${version}-arm64.zip" > SHA256SUMS)
printf 'Packages: %s\n' "$out"
