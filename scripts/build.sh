#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
app="$PWD/build/Voice Pill.app"
bash scripts/build-icon.sh
if [ ! -x build/freeasr ]; then bash scripts/build-freeasr.sh; fi
bash scripts/build-codex-asr.sh
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources" build/module-cache
xcrun swiftc -parse-as-library -swift-version 5 -O -target arm64-apple-macosx26.0 -module-cache-path "$PWD/build/module-cache" Sources/*.swift -o "$app/Contents/MacOS/VoicePill"
cp Resources/Info.plist "$app/Contents/Info.plist"
cp Resources/AppIcon.icns "$app/Contents/Resources/AppIcon.icns"
cp Resources/transcribe-audio "$app/Contents/Resources/transcribe-audio"
cp Resources/*LICENSE "$app/Contents/Resources/"
cp LICENSE "$app/Contents/Resources/VoicePill-LICENSE"
cp build/freeasr "$app/Contents/Resources/freeasr"
cp build/codex-asr "$app/Contents/Resources/codex-asr"
python3 scripts/bundle-libraries.py "$app"
python3 scripts/collect-licenses.py "$app"
if [ "${VOICEPILL_DISTRIBUTION:-0}" = 1 ]; then
  python3 scripts/sign-release.py "$app"
else
  python3 scripts/sign.py "$app"
fi
codesign --verify --deep --strict "$app"
printf '%s\n' "$app"
