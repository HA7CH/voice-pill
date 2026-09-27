# FreeASR

Free speech-to-text CLI tool and OpenAI-compatible API server, powered by reverse-engineered DoubaoIME ASR.

**No API key required.** Device registration is automatic.

## Features

- **CLI transcription**: `freeasr transcribe audio.wav`
- **HTTP server**: OpenAI-compatible `POST /v1/audio/transcriptions`
- **Multiple formats**: text, JSON, verbose JSON, SRT, VTT
- **Streaming**: supports any audio format via ffmpeg (wav, mp3, m4a, flac, video, etc.)
- **Zero config**: automatic device registration, credential caching

## Installation

### Prerequisites

- **libopus** and **libopusfile**: required for Opus audio encoding (cgo)
- **ffmpeg**: required for audio format conversion

```bash
# Debian/Ubuntu
sudo apt install libopus-dev libopusfile-dev ffmpeg

# macOS
brew install opus opusfile ffmpeg

# Arch Linux
sudo pacman -S opus opusfile ffmpeg
```

### Build from source

```bash
git clone https://github.com/WEIFENG2333/FreeASR.git
cd FreeASR
go build -o freeasr .
```

### Download binary

See [Releases](https://github.com/WEIFENG2333/FreeASR/releases).

## Usage

### Transcribe audio file

```bash
# Basic usage — outputs plain text
freeasr transcribe recording.wav

# JSON output (OpenAI-compatible)
freeasr transcribe recording.mp3 --format json

# Generate SRT subtitles
freeasr transcribe video.mp4 --format srt -o subtitles.srt

# Generate WebVTT subtitles
freeasr transcribe podcast.m4a --format vtt

# Verbose JSON with segments
freeasr transcribe audio.wav --format verbose_json

# Read from stdin (pipe from ffmpeg)
ffmpeg -i video.mp4 -f wav - | freeasr transcribe -

# With realtime pacing (more stable for large files)
freeasr transcribe long_audio.wav --realtime
```

### Start HTTP server

```bash
# Start server on default port (8080)
freeasr serve

# Custom port with auth
freeasr serve --port 9090 --auth-token my-secret-key

# Server endpoints:
#   POST /v1/audio/transcriptions  — transcribe audio file
#   GET  /v1/models               — list models
#   GET  /health                   — health check
```

#### Use with curl

```bash
# Text response
curl -X POST http://localhost:8080/v1/audio/transcriptions \
  -F file=@audio.wav \
  -F response_format=text

# JSON response
curl -X POST http://localhost:8080/v1/audio/transcriptions \
  -F file=@audio.wav \
  -F model=freeasr-1 \
  -F response_format=json
```

#### Use with OpenAI Python SDK

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:8080/v1", api_key="any")

with open("audio.wav", "rb") as f:
    result = client.audio.transcriptions.create(
        model="freeasr-1",
        file=f,
        response_format="json",
    )
    print(result.text)
```

### System check

```bash
freeasr doctor    # Check ffmpeg, libopus, credentials, connectivity
freeasr auth      # Manually register device and get token
freeasr version   # Print version info
```

## API Compatibility

| OpenAI Parameter     | Supported | Notes                           |
|---------------------|-----------|---------------------------------|
| `file`              | ✅        | Required, multipart upload      |
| `model`             | ✅        | Accepted but ignored            |
| `response_format`   | ✅        | text/json/verbose_json/srt/vtt  |
| `language`          | ✅        | Accepted but not used by server |
| `prompt`            | ❌        | Not supported                   |
| `temperature`       | ❌        | Not supported                   |
| `timestamp_granularities` | ❌  | Not supported                   |

## Configuration

### Global flags

```
--credential-path <path>   Credential cache file (default: ~/.config/freeasr/credentials.json)
--log-level <level>        Log level: error|warn|info|debug
-v                         Debug logging
-q                         Quiet mode (errors only)
```

### Environment variables

| Variable                  | Description              |
|--------------------------|--------------------------|
| `FREEASR_CREDENTIAL_PATH` | Credential file path    |

## Disclaimer

- This project uses **unofficial, reverse-engineered APIs** from DoubaoIME (豆包输入法)
- **For learning and research purposes only** — not for commercial use
- Service availability may change at any time without notice
- The API may be rate-limited or blocked

## License

MIT
