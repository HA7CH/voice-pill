#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
export SDKROOT="$(xcrun --sdk macosx --show-sdk-path)"
export CC="$(xcrun --find clang)"
export CGO_ENABLED=1
export MACOSX_DEPLOYMENT_TARGET=26.0
mkdir -p build
cd vendor/FreeASR
go build -tags nolibopusfile -trimpath -ldflags="-s -w" -o ../../build/freeasr .
