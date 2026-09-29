# Codex streaming dictation (experimental)

This local `0.1.3-stream.1` build adds the protocol observed in Codex Desktop
26.924.22138. It is not an upstream release or the public OpenAI Realtime API.
The existing `transcribe` and `serve` commands still use HTTP file transcription.

## Usage

`stream` consumes **raw signed PCM16 little endian, mono**, not a WAV/M4A
container. Set `--sample-rate` to the actual input rate (default 24000 Hz).
It neither captures the microphone nor performs resampling itself.

```sh
# Convert an authorized test recording locally, then play it to the service
# at real-time speed. --input automatically paces file input.
ffmpeg -i test.wav -ar 24000 -ac 1 -f s16le test.pcm
codex-asr stream --input test.pcm --sample-rate 24000 --language zh

# A live recorder can send PCM through stdin. Closing stdin finishes dictation.
# For file-based pipe testing, -re paces ffmpeg output.
ffmpeg -re -i test.wav -ar 24000 -ac 1 -f s16le pipe:1 |
  codex-asr stream --sample-rate 24000 --language zh
```

Authentication follows the existing CLI: explicit `--auth-file`,
`CODEX_ASR_BEARER`, or `$CODEX_HOME/auth.json` / `~/.codex/auth.json`.
Only the access token and account ID are sent to ChatGPT over TLS. The command
does not modify or refresh login credentials. Expired credentials require
signing in again through Codex. Do not put tokens in command-line arguments.

Default endpoint:
`wss://chatgpt.com/backend-api/dictation/stream?dictation_surface=composer`.
Streaming supports direct connections and HTTP CONNECT proxies through the
existing proxy settings. Unsupported proxy schemes fail explicitly. Workspace
specific backend routing is not implemented in this experimental adapter.

## Output contract

Stdout is flushed JSON Lines. Every line has `elapsed_ms` since the WebSocket
session began (it excludes connection setup):

- `ready`: the server accepted `session.start`; includes `sample_rate`.
- `partial`: current **full combined text** in `text`, plus `utterance_id` and
  `revision`. Replace the displayed text; do not append it as a delta.
- `final`: one utterance finalized; `text` is still the combined transcript.
  Further utterances may follow, so this does not mean recording ended.
- `input.finished`: EOF received and all input sent; includes `audio_bytes`.
- `result`: every observed utterance finalized and the server acknowledged
  session closure. This is the only completed full transcript.

Errors go to stderr and exit nonzero. A disconnect, timeout, failed utterance,
or close with partial-only text never emits a successful `result`. If the server closes a silent utterance without a final transcript, the command
exits nonzero instead of reporting a successful transcription. Ctrl+C cancels without emitting a completed result.
Applications must retain their own audio file for manual retry or HTTP fallback;
this command does not save audio and does not automatically replay live input.

The client requests a 30-second utterance boundary and a five-minute session.
An utterance boundary does not end the recording; later utterances are assembled
in speech-start order. There is no session rollover yet: callers should stop
before five minutes and retain audio on timeout. The finalization timeout is
60 seconds, allowing for the server's observed 55-second provider close timeout.

## Build and verification

```sh
cargo test --all-features
cargo clippy --all-targets --all-features -- -D warnings
cargo check --no-default-features --lib --bin codex-asr
cargo build --release --locked
```

The `streaming` feature is enabled by default and can be built independently
with `--no-default-features --features streaming`. Offline tests cover revision
replacement, late partials, out-of-order finalization, truncated PCM samples,
and close acknowledgment without final text. Network tests should only use
audio authorized for upload; synthetic speech is sufficient.

Voice Pill 0.2 connects its capture pipeline to `stream` and displays partial
text snapshots. It retains full audio locally for file-based retry after failure.
