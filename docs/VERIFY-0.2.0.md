# Voice Pill 0.2.0 local verification

Validated 2026-09-30 on the existing Apple Silicon Mac. The running local build
keeps its original bundle identifier and signing identity. Public build retains
its separate identity and contains no login files or personal recordings.

| Check | Result |
| --- | --- |
| Codex real WebSocket through Swift transport | Passed; synthetic Chinese speech, 43 caption updates, full result after EOF |
| Doubao real WebSocket through same transport | Passed; synthetic Chinese speech, 21 caption updates, full result after EOF |
| Provider selector | Changed to Doubao and back to Codex through UI; preference persisted |
| Current default | Codex; Fn and alternate shortcut use selected provider, Ctrl+Fn uses Codex |
| Fragmented JSON / UTF-8 and numeric metadata | Passed |
| Codex utterance final vs session result | Passed; utterance final alone cannot complete the session |
| Child disconnect before input EOF | Passed; error returned without waiting for mic/input to stop |
| Existing bridge and FreeASR tests | Passed |
| Local editable target | Two insertion checks passed; target UI readback contained both test sentences |
| Local Accessibility identity | Same designated requirement; access remains granted |
| Existing login item | Still targets the same updated local app path |
| Public bundle | Signature, portable dylibs, both helper launches and private-file scan passed |
| Actual hold-Fn microphone interaction | Not exercised; tests used synthetic audio to avoid ambient capture |
| Full microphone recording after a network loss and Retry | Retention/fallback code preserved; real mic/network interruption not exercised |
| Third-party app input fields | Not exercised this turn |

The Codex first-caption latency was about 4.95 seconds including connection setup
in this run; Doubao was about 1.65 seconds. These are individual measurements,
not a latency guarantee. No private historical audio was uploaded. Glass and
caption animation source files were not modified.
