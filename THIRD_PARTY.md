# Third-party notices

Voice Pill's original code is MIT, © 2026 HA7CH. Upstream components keep their original notices and licenses.

| Component | Source / revision | License / use |
| --- | --- | --- |
| LiquidType HUD, glass, captions, waveform | https://github.com/LuliYanng/LiquidType/tree/03d5659503923346eed7d36f25ddf4831b606b58 | MIT; full notice in `Resources/LiquidType-LICENSE` |
| PasteController adaptation | https://github.com/simicvm/whisper/tree/45a497caa5cfb8b78a5e322343c7a615bd1e3052 | MIT; full notice in `Resources/Whisper-LICENSE` |
| ANC audio bridge | HA7CH | MIT; full notice in `Resources/ANC-Transcribe-Audio-LICENSE` |
| FreeASR | https://github.com/WEIFENG2333/FreeASR/tree/252f6387a487d7f9993b78ff0302c1a6d5c58f59 | Upstream README declares MIT; preserved in `vendor/FreeASR/README.md`. Upstream also describes its unofficial service integration as learning/research only, not commercial use. This is not a grant of access to Doubao's service. |
| libopus | https://opus-codec.org/ | BSD-style; full COPYING is included with packaged libraries |
| Codex ASR streaming fork | https://github.com/Wangnov/codex-asr/tree/479f6a7 + local streaming implementation | MIT; `Resources/CodexASR-LICENSE`; complete source in `vendor/codex-asr/` |
| Rust crates | `vendor/codex-asr/Cargo.lock` | LICENSE/COPYING/NOTICE files and license expressions collected into the app’s `ThirdPartyLicenses/` |
| Go modules | `vendor/FreeASR/go.mod` and `go.sum` | Original LICENSE/COPYING files collected into the app's `Contents/Resources/ThirdPartyLicenses/` |

Our FreeASR changes add streaming stdin PCM, JSON-line captions, partial-result handling, bounded finalization, local transport tests, and direct reading of Voice Pill's PCM WAV files without ffmpeg. This snapshot is source code, not an embedded Git repository.

The distribution does not include WeType/dicta-asr binaries, Codex credentials, FFmpeg, or developer signing keys. The Codex adapter is bundled; the user signs into Codex separately. Voice Pill is not affiliated with Apple, ByteDance, Tencent, or OpenAI.
