#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
export SDKROOT="$(xcrun --sdk macosx --show-sdk-path)"
export CC="$(xcrun --find clang)"
export MACOSX_DEPLOYMENT_TARGET=26.0
bash scripts/test-bridge.sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
swiftc Sources/DeliverySession.swift tests/DeliverySessionTests.swift -o "$work/delivery-test"
"$work/delivery-test"
swiftc Sources/StreamPipe.swift tests/StreamPipeTests.swift -o "$work/stream-test"
"$work/stream-test"
swiftc Sources/LiveProtocol.swift tests/LiveProtocolTests.swift -o "$work/protocol-test"
"$work/protocol-test"
swiftc Sources/LiveProtocol.swift Sources/LiveTranscriptionSession.swift Sources/StreamPipe.swift tests/LiveSessionTests.swift -o "$work/session-test"
"$work/session-test"
(cd vendor/FreeASR && go test -tags nolibopusfile ./...)
