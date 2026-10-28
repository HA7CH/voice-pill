# macOS distribution

The public application identity is `com.ha7ch.voicepill`. Keep its Developer ID team stable between releases. Never upload `.signing`, keychains, certificate private keys, account credentials, or recordings.

## First-time setup on a release Mac

1. Install a valid **Developer ID Application** certificate and its private key in a usable keychain. An Apple Development or Apple Distribution certificate is not a substitute. A `.cer` file alone has no private key. If a legacy keychain cannot be unlocked, preserve it and ask the Account Holder for a new Developer ID certificate; do not revoke existing certificates unnecessarily.
2. Configure `notarytool store-credentials` interactively with your Apple developer authentication, using the profile name `voice-pill`. Do not put passwords in shell commands, source, or chat.
3. Create an isolated Python environment, then install `ds-store==1.3.3` for Finder metadata. By default the packager uses `build/dmg-tools/bin/python`; override with `VOICEPILL_DMG_PYTHON`.

## Release

Set `VOICEPILL_DEVELOPER_ID` to the certificate identity or SHA-1 fingerprint, `VOICEPILL_NOTARY_PROFILE` to the saved profile name, and optionally `VOICEPILL_SIGNING_KEYCHAIN` to its keychain path. Run `bash scripts/release.sh`.

The pipeline signs all bundled Mach-O helpers and libraries with hardened runtime and secure timestamps, signs the app with microphone entitlement, notarizes and staples the app, verifies Gatekeeper, creates the two-icon DMG, signs/notarizes/staples the DMG, verifies Gatekeeper again, and only then writes final checksums. The ZIP contains the stapled app. Any failed gate stops release; there is no ad-hoc fallback.

`bash scripts/package.sh --preview` creates a layout-only preview in `build/installer-preview/`. It is not a public release and does not assert Gatekeeper acceptance.

Before publishing, mount the DMG and inspect the real Finder window: Voice Pill on the left, Applications on the right, 160-point icons, drag arrow, no README on the installation surface. Test a quarantined downloaded copy and actual app launch, recording permissions, Fn capture, and destination insertion independently. A duplicate-instance launch exiting successfully does not prove the main app launched.

## Xcode cloud signing alternative

A logged-in Xcode account with cloud-managed Developer ID permission can sign and upload without unlocking an old local private key. After building the app, run `python3 scripts/cloud-release.py --team-id YOUR_TEAM_ID`. Preserve the archive while Apple processes it, then run the same command with `--export-notarized`. Export, staple validation, and Gatekeeper must all succeed before packaging promotes the app to `dist/`.

In this path the **app** is Developer ID-signed, notarized and stapled; the custom DMG container is unsigned unless a usable local `VOICEPILL_DEVELOPER_ID` is also supplied. Do not describe the container itself as signed or notarized. Verify the app extracted from the final DMG independently.
