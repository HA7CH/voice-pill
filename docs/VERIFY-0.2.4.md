# Voice Pill 0.2.4 verification

Checked on 2026-10-02 on Apple Silicon / macOS 27.

The running 0.2.3 app was busy with no transcription child, and ignored Fn. Accessibility remained granted. Code review found an insertion path that returned after focus loss without leaving the inserting phase.

0.2.4 turns focus loss into a recoverable paste failure. Every unfinished insertion returns to a nonbusy state; an eight-second watchdog invalidates a stalled attempt. Stale callbacks cannot modify a newer session. The last transcript remains available in settings. Unfinished audio remains available for Retry across restart.

- Regression tests passed: early return unlocks; successful completion stays successful; timeout fires once; a newer session is unaffected.
- Codex / Doubao protocol, streaming-session, disconnection and audio-bridge tests passed.
- Local build passed, original signing requirement retained, Accessibility granted, provider selector enabled after restart.
- Phase transitions now produce bounded diagnostic logs containing state, session IDs and timing, without speech text.

Physical Fn and third-party paste acceptance are separate from these automated checks. No ambient microphone recording was made by the verification process.
