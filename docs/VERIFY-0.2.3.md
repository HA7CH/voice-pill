# Voice Pill 0.2.3 release verification

Validated on 2026-10-02 on Apple Silicon, with macOS 27 and a macOS 26 deployment target.

- Swift build and bundled Codex ASR / FreeASR helpers passed.
- Audio bridge, fragmented UTF-8 / JSON events, Codex and Doubao transport sessions, and disconnect handling tests passed.
- Bundle signature, portable library dependencies, helper launch and private-artifact checks passed.
- DMG integrity and both package SHA256 checks passed. The ZIP contains version 0.2.3, build 5, with the public bundle identifier `com.ha7ch.voicepill`.
- Captions and completed transcripts use the selected provider's output directly. No local correction worker, model weights or Python inference runtime is bundled.
- LiquidType UI, provider switching, saved-recording Retry and direct insertion behavior are retained.

This release check used synthetic transport tests, without recording ambient audio. Real microphone dictation and third-party text fields were not retested for this release. The fresh post-removal input fixture from the preceding local update could not acquire focus, so it is not counted as a passing insertion check. Earlier live-provider and insertion checks are recorded separately in `VERIFY-0.2.0.md`.

Public downloads use ad-hoc signing and are not Apple-notarized. The existing private local development installation keeps its original signing identity; it is not uploaded.
