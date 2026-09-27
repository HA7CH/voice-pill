#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
app="$PWD/build/Voice Pill.app"
if [ ! -x build/freeasr ]; then bash scripts/build-freeasr.sh; fi
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources" build/module-cache
xcrun swiftc -parse-as-library -swift-version 5 -O -target arm64-apple-macosx26.0 -module-cache-path "$PWD/build/module-cache" Sources/*.swift -o "$app/Contents/MacOS/VoicePill"
cp Resources/Info.plist "$app/Contents/Info.plist"
cp Resources/transcribe-audio "$app/Contents/Resources/transcribe-audio"
cp Resources/*LICENSE "$app/Contents/Resources/"
cp LICENSE "$app/Contents/Resources/VoicePill-LICENSE"
cp build/freeasr "$app/Contents/Resources/freeasr"
python3 scripts/bundle-libraries.py "$app"
python3 scripts/collect-licenses.py "$app"
python3 scripts/sign.py "$app"
codesign --verify --deep --strict "$app"
printf '%s\n' "$app"
