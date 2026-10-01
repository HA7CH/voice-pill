# Voice Pill

Native macOS hold-to-talk dictation with Liquid Glass, live captions and automatic insertion into the original app. No menu-bar or Dock icon.

Version **0.2.3** supports switchable Codex / Doubao live transcription. Captions and pasted text use the provider's output directly, without installing or running a local correction model.

## Install

Apple Silicon, macOS 26+, internet required. This initial build was checked on macOS 27; real microphone behavior on macOS 26 is not yet verified.

Download the DMG from [Releases](https://github.com/HA7CH/voice-pill/releases/latest), drag **Voice Pill.app** into Applications, eject the disk and launch the installed app. Enable Accessibility and Microphone, accept the selected service's upload prompt, then restart after granting Accessibility. Set the globe key to **Do Nothing** in System Settings → Keyboard.

This preview is ad-hoc signed, **not Apple-notarized**. If blocked, verify the source and use Privacy & Security → Open Anyway. Do not disable Gatekeeper globally. Updates may require regranting Accessibility.

Place the cursor in a text field. Hold **Fn**, speak, release to paste. Codex and Doubao streaming backends are bundled. **Speech provider** switches the Fn and alternate-shortcut provider; the default is Codex, which requires an existing Codex ChatGPT login. **Ctrl+Fn** always uses Codex live captions. No Homebrew, FFmpeg or API key is needed. **Esc** cancels. **Ctrl+Option+Space** toggles recording. Reopen the app for settings, saved recordings / Retry, or Quit.

Audio goes directly to your selected third-party service after consent. Doubao uses FreeASR's unofficial IME protocol, described upstream as research/learning use; availability is not guaranteed. This is not offline transcription. Failed audio remains locally for retry; successful or canceled recordings are removed. No message is automatically sent by pressing Return. See [privacy](PRIVACY.md).

## Build

Install a macOS 26+ SDK toolchain, Python 3, Go 1.25+, Rust stable, Homebrew `opus` and `pkg-config`. Run `bash scripts/test.sh`, `bash scripts/build.sh`, then `bash scripts/package.sh`. The build bundles dylibs and license notices. The project is MIT with third-party notices in [THIRD_PARTY.md](../THIRD_PARTY.md). LiquidType supplies the HUD implementation; FreeASR supplies the modified streaming backend.
