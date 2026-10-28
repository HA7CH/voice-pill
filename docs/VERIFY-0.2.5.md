# Voice Pill 0.2.5 / build 7 verification

- App identifier: `com.ha7ch.voicepill`; Apple Silicon, macOS 26+.
- Developer ID Application signature, team V8DZ785G5L, secure timestamp and hardened runtime verified. Xcode automatic distribution used its authenticated cloud-managed signing path; no legacy private key was exported or reset.
- Xcode notarization upload completed; `-exportNotarizedApp` succeeded. `stapler validate` passed. Gatekeeper returned `accepted`, source `Notarized Developer ID`.
- Final compressed DMG verified by `hdiutil verify`. Its two visible items are Voice Pill.app and an Applications symlink. Finder's actual window was inspected: 160-point icons, app left, Applications right, drag arrow, no README on the installation surface.
- A new app extracted from the final DMG, marked with a download quarantine attribute, passed deep strict signature validation, stapled ticket validation, and Gatekeeper assessment. Portable libraries, helper launch, licenses, and absence of private artifacts passed `verify-bundle.py`.
- The custom DMG container is unsigned in the cloud-signing path; the app inside is signed, notarized and stapled. Do not conflate the two.
- Same-ID duplicate launches were tested three times; the public-ID binary also declined to start beside the original local-ID app. One shortcut owner remained.
- This packaging/signing verification does not prove microphone capture or pasted text acceptance. Those require a physical Fn session and destination-field confirmation, and Accessibility authorization for the final signing identity.
