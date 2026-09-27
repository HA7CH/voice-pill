#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
export SDKROOT="$(xcrun --sdk macosx --show-sdk-path)"
export CC="$(xcrun --find clang)"
export MACOSX_DEPLOYMENT_TARGET=26.0
bash scripts/test-bridge.sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
swiftc Sources/StreamPipe.swift tests/StreamPipeTests.swift -o "$work/stream-test"
"$work/stream-test"
(cd vendor/FreeASR && go test -tags nolibopusfile ./...)
