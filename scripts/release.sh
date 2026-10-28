#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
: "${VOICEPILL_DEVELOPER_ID:?Select a usable Developer ID Application identity}"
: "${VOICEPILL_NOTARY_PROFILE:?Configure a notarytool keychain profile first}"
export VOICEPILL_DISTRIBUTION=1
bash scripts/build.sh
python3 scripts/verify-bundle.py 'build/Voice Pill.app'
python3 scripts/notarize.py 'build/Voice Pill.app'
spctl --assess --type execute --verbose=4 'build/Voice Pill.app'
bash scripts/package.sh
version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' 'build/Voice Pill.app/Contents/Info.plist')
python3 scripts/notarize.py "dist/Voice-Pill-${version}-arm64.dmg"
spctl --assess --type open --context context:primary-signature --verbose=4 "dist/Voice-Pill-${version}-arm64.dmg"
(cd dist && shasum -a 256 "Voice-Pill-${version}-arm64.dmg" "Voice-Pill-${version}-arm64.zip" > SHA256SUMS)
printf 'Verified notarized release: %s\n' "$version"
