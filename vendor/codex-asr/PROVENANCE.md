# Codex ASR streaming adapter

Upstream: https://github.com/Wangnov/codex-asr (MIT), commit `479f6a7`.
Local streaming implementation: `0.1.3-stream.1`, commit `63f3f1b` in the local codex-asr checkout.

The complete buildable Rust sources and lockfile are included here. Voice Pill
builds the CLI with the `streaming` feature and without the unused REST server.
The adapter implements the dictation WebSocket protocol observed in Codex Desktop
26.924.22138. It reuses local Codex authentication. No credentials or recordings
are included. This is an experimental fork, not an upstream release.
