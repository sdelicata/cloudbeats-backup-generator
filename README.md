# CloudBeats Backup Generator

A CLI tool that generates `.cbbackup` files for the [CloudBeats](https://play.google.com/store/apps/details?id=com.cointail.cloudbeats) Android music player by scanning a local Dropbox-synced music folder, reading audio tags, and fetching file IDs from the Dropbox API.

## Overview

CloudBeats reads music from Dropbox but its library scanner is unreliable — it doesn't index all files. However, the app supports importing a `.cbbackup` file containing the full library. This tool generates that file externally so you always have a complete, up-to-date library.

It reads audio metadata (artist, album, duration, etc.) directly from local files and only uses the Dropbox API to retrieve account and file identifiers needed by CloudBeats.

**Supported audio formats:** MP3, M4A, FLAC, OGG, Opus, WAV, WMA, AAC, DSF, AIFF, AIF, APE, WavPack, Musepack.

## Prerequisites

- **Go 1.24+**
- **TagLib** — native C++ audio metadata library
  ```sh
  brew install taglib   # macOS
  ```
- **Dropbox Desktop** installed and syncing the music folder locally
- **Dropbox API credentials** (see below)

## Dropbox Authentication

The tool supports two authentication methods:

### Method 1: Interactive Setup (Recommended)

Uses long-lived credentials to automatically obtain a fresh access token at each run. On first run, the tool detects missing credentials and launches an interactive OAuth2 setup: it prompts for your app key and secret, opens the authorization URL in your browser, exchanges the code, and stores credentials locally. Subsequent runs need zero auth flags.

**App setup** (one-time):

1. Go to <https://www.dropbox.com/developers/apps>
2. Click **Create app**
3. Choose **Scoped access** and **Full Dropbox** access
4. Name your app (e.g. `cloudbeats-backup`) and click **Create app**
5. Go to the **Permissions** tab and enable:
   - `files.metadata.read`
   - `account_info.read`
6. Click **Submit** to save the permissions
7. Note your **App key** and **App secret** from the **Settings** tab

**First run** — the tool prompts for credentials automatically:

```sh
./cloudbeats-backup-generator --local ~/Dropbox/Music
# → prompts for app key and app secret, opens browser for authorization
```

**Subsequent runs** (no auth flags needed):

```sh
./cloudbeats-backup-generator --local ~/Dropbox/Music
```

Credentials are stored in `~/Library/Application Support/cloudbeats-backup-generator/credentials.json` (macOS).

You can also provide credentials explicitly via flags or environment variables for CI/scripting:

```sh
./cloudbeats-backup-generator --local ~/Dropbox/Music \
  --app-key YOUR_APP_KEY \
  --app-secret YOUR_APP_SECRET \
  --refresh-token YOUR_REFRESH_TOKEN
```

### Method 2: Short-Lived Token

Uses a manually generated access token that expires after ~4 hours.

1. Follow the **App setup** steps above
2. Go to the **Settings** tab
3. Under **OAuth 2 > Generated access token**, click **Generate**
4. Copy the token

```sh
./cloudbeats-backup-generator --local ~/Dropbox/Music --token "sl.xxxxx"
```

> **Note:** Developer console tokens expire after ~4 hours. You'll need to generate a new one each time you run the tool.

## Installation

```sh
# Clone and build
git clone https://github.com/sdelicata/cloudbeats-backup-generator.git
cd cloudbeats-backup-generator
make build
```

Or directly with Go:

```sh
go build -o cloudbeats-backup-generator ./cmd
```

## Usage

```
cloudbeats-backup-generator [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--local` | *(required)* | Path to the local folder to scan (must be inside the Dropbox folder) |
| `--output` | `cloudbeats.cbbackup` | Path to the output `.cbbackup` file |
| `--app-key` | | Dropbox app key (also read from `DROPBOX_APP_KEY` env var) |
| `--app-secret` | | Dropbox app secret (also read from `DROPBOX_APP_SECRET` env var) |
| `--refresh-token` | | Dropbox refresh token (also read from `DROPBOX_REFRESH_TOKEN` env var) |
| `--token` | | Dropbox short-lived access token (also read from `DROPBOX_TOKEN` env var) |
| `--workers` | `0` (auto: 2x CPU cores) | Number of parallel workers for reading audio tags |
| `--dry-run` | `false` | Show Dropbox mapping without reading tags or writing a file |
| `--log-level` | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error` |

**Token resolution priority:**
1. Explicit flags (`--app-key` + `--app-secret` + `--refresh-token`)
2. Stored credentials (if all fields present)
3. Direct token (`--token` / `DROPBOX_TOKEN`)
4. Interactive setup (prompts on first run if terminal is interactive)

Each flag falls back to its corresponding environment variable.

### Examples

```sh
# First run: prompts interactively for app key, app secret, and authorization
./cloudbeats-backup-generator --local ~/Dropbox/Music

# Subsequent runs: uses stored credentials automatically
./cloudbeats-backup-generator --local ~/Dropbox/Music

# Using explicit refresh token (for CI/scripting)
./cloudbeats-backup-generator --local ~/Dropbox/Music \
  --app-key "abc123" --app-secret "xyz789" --refresh-token "def456"

# Using a short-lived access token
./cloudbeats-backup-generator --local ~/Dropbox/Music --token "sl.xxxxx"

# Custom output path
./cloudbeats-backup-generator --local ~/Dropbox/Music --output ~/Desktop/backup.cbbackup

# Dry run — validate the Dropbox mapping without writing anything
./cloudbeats-backup-generator --local ~/Dropbox/Music --dry-run

# Verbose logging
./cloudbeats-backup-generator --local ~/Dropbox/Music --log-level debug
```

## How It Works

1. **Authenticate** — Obtains a fresh access token (via refresh token) or uses the provided token, then retrieves your account ID
2. **Scan & match** — Detects the Dropbox root path from `~/.dropbox/info.json`, scans the local folder for audio files, lists the corresponding Dropbox folder via the API, and matches local files to their Dropbox entries (case-insensitive, NFC-normalized)
3. **Read tags** — Reads ID3/audio metadata (title, artist, album, duration, etc.) from each local file using a parallel worker pool
4. **Build backup** — Assembles each matched file into a `.cbbackup` item with its Dropbox file ID and audio metadata
5. **Write file** — Serializes to JSON and writes the `.cbbackup` file

## Importing into CloudBeats

1. Transfer the generated `.cbbackup` file to your Android device
2. Open CloudBeats
3. Go to **Settings > Backup & Restore > Restore**
4. Select the `.cbbackup` file
5. Your full library will be imported

Playlists are not included in the generated backup — you can recreate them in the app.
