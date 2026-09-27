# Privacy and local data

Voice Pill records only after its recording shortcut or explicit microphone setup action. The nonactivating HUD leaves keyboard focus in your original app.

- **Doubao:** microphone PCM is sent through the unofficial Doubao IME protocol, using device registration/settings endpoints from ByteDance and its ASR WebSocket. Registration creates synthetic device identifiers stored locally. No paid API key is configured. The provider controls remote processing and retention; Voice Pill cannot promise remote deletion.
- **Codex (optional):** a separately installed `codex-asr` reads your local Codex sign-in and sends the audio to ChatGPT's transcription endpoint. Voice Pill does not ship or upload your Codex credentials to HA7CH.
- **HA7CH:** no speech proxy, cloud storage, application analytics, or automatic crash-report uploads are implemented.
- **Accessibility:** used to observe the dictation shortcut, remember the destination control, paste text and check insertion. The clipboard is temporarily changed during paste and restored when safe. Some apps cannot expose their text for verification.

Data under `~/Library/Application Support/VoicePill/`:

- `Recordings/`: failed/unfinished WAV or M4A recordings and backend sidecars, retained across restarts for explicit Retry. Success or explicit cancel removes the current audio. Short-lived output/error files may exist during transcription; inspect this folder before sharing diagnostics.
- `doubao-credentials.json`: device identifiers and service token. Never attach this file to an issue.

The latest transcript is held in memory, not a persistent transcript database. If pasting fails after successful transcription, copy the text before quitting. Bounded temporary event logs record session IDs, timing and phase transitions, without intended speech content; error logs from third-party programs may contain diagnostic details. macOS stores preferences and permission records separately.

To remove the app, quit from its settings and move the installed app to Trash. Local recordings and credentials are intentionally not deleted by uninstalling; remove the VoicePill Application Support folder yourself if desired, after recovering anything you need. Remove its microphone/accessibility entries in System Settings if desired.
