#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
printf 'fixture' > "$work/input.m4a"
cat > "$work/asr" <<'MOCK'
#!/bin/bash
printf '这是原话，嗯，不做改写。\n'
MOCK
chmod +x "$work/asr"
CODEX_ASR_BIN="$work/asr" bash Resources/transcribe-audio "$work/input.m4a" '' "$work/result.md" >/dev/null
grep -q '这是原话，嗯，不做改写。' "$work/result.md"
if CODEX_ASR_BIN="$work/asr" bash Resources/transcribe-audio "$work/input.m4a" '' "$work/result.md" 2>/dev/null; then exit 1; fi
cat > "$work/asr" <<'MOCK'
#!/bin/bash
printf 'partial text that must not be delivered'
printf 'upstream failed' >&2
exit 1
MOCK
if CODEX_ASR_BIN="$work/asr" TRANSCRIBE_AUDIO_MAX_ATTEMPTS=1 bash Resources/transcribe-audio "$work/input.m4a" '' "$work/failed.md" 2> "$work/error"; then exit 1; fi
test ! -e "$work/failed.md"
grep -q 'upstream failed' "$work/error"
printf 'PASS: faithful output, overwrite protection, no partial artifact on failure\n'
