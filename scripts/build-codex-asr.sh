#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
export CARGO_TARGET_DIR="$PWD/build/codex-asr-target"
export MACOSX_DEPLOYMENT_TARGET=26.0
cargo build --manifest-path vendor/codex-asr/Cargo.toml --release --locked --no-default-features --features streaming
cp "$CARGO_TARGET_DIR/release/codex-asr" build/codex-asr
