<div align="center">

```
╭░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░╮
│ (o)  ╭───╮                                                         (o) │
│      │ B │  ──────────────────────────────────────────  [ ]IN [ ]OUT   │
│      ╰───╯                                                             │
│  ╭──────────────────────────────────────────────────────────────────╮  │
│  │                  CASSETTE • RETRO SPOTIFY TUI                    │  │
│  │          ▄███▄          ╭─────────────╮          ▄███▄           │  │
│  │         █▀ █ ▀█         │███ │ │ │  █ │         █▀ █ ▀█          │  │
│  │        █ ▄ █ ▄ █        │███ │ │ │  █ │        █ ▄ █ ▄ █         │  │
│  │         █▄ █ ▄█         │███ │ │ │  █ │         █▄ █ ▄█          │  │
│  │          ▀███▀          ╰─────────────╯          ▀███▀           │  │
│  ├──────────────────────────────────────────────────────────────────┤  │
│  │          01:23 [████████────────] 03:45   ▶ PLAYING ⇌           │  │
│  ╰──────────────────────────────────────────────────────────────────╯  │
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│    ╲   ( )                                                  ( )   ╱    │
│(o)  ╲         ( )                                    ( )         ╱  (o)│
╰──────╲──────────────────────────────────────────────────────────╱──────╯
```

# cassette

**A high-fidelity, retro ANSI animated cassette tape music player for Spotify.**  
*Because every other Spotify client is either a 500MB Electron dumpster fire or looks like an Excel spreadsheet from 1994.*

[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux-blue.svg)](https://github.com)

</div>

---

## Why does this exist? (The Rant)

Every modern Spotify client has completely lost the plot:

1. **The Official Spotify Desktop App**: Consumes 2GB of RAM, renders web views inside web views, constantly shoves podcasts and unskippable clutter down your throat, and takes 10 seconds just to open a search bar.
2. **Generic "Terminal" Spotify TUIs**: Most CLI players are dead, broken, abandoned, or sterile. They render lifeless white tables that look like a corporate tax audit. No style, no warmth, no joy.
3. **The "Play on Device" Nightmare**: Half the terminal clients out there don't even play sound. You press play, and nothing happens because you have to spend 4 hours configuring external daemon sockets, broken D-Bus hooks, and third-party audio servers.

**Enough.**

**cassette** was built with a simple philosophy: **Music should feel like physical media again.**
You get a handcrafted, pixel-aligned ANSI cassette tape that physically spins in real time inside your terminal. It connects directly to Spotify, features automated background playback with an integrated `librespot` engine, auto-queues endless song radio, and stays lightning fast.

---

## Features

- **Animated ANSI Cassette Art**: Realistic rotating 6-tooth gear sprockets, calibrated tape window, dynamic supply/take-up spools that transfer tape in real time, and HUD metadata.
- **Standalone Terminal Audio**: Bundled with native `librespot` Spotify Connect support. No official Spotify desktop client needed. Your PC shows up natively as `cassette`.
- **Endless Song Radio**: Playing any track from search automatically seeds Spotify's similar-track recommendation engine, queueing endless continuous playback so the music never stops.
- **Distraction-Free Keybindings**: Full keyboard controls for search, playlists, tracks, albums, devices, volume, seeking, and shuffle.
- **Zero Electron Bloat**: Pure Go + Bubble Tea + Lipgloss. Fast startup, minimal CPU usage, and low memory footprint.

---

## Requirements

- **Spotify Premium** account (required by Spotify for Web API playback & Spotify Connect).
- **Linux** (PulseAudio / PipeWire).
- **librespot** (installed on Arch via `pacman -S librespot`, or auto-detected by `cassette`).
- Standard terminal with UTF-8 support (Kitty, Konsole, Alacritty, WezTerm, iTerm2, etc.).

---

## Quick Install

### One-Command Installer

Clone the repo and run the automated installer:

```bash
git clone https://github.com/OreoMuncher45/cassette.git
cd cassette
chmod +x install.sh && ./install.sh
```

This compiles the binary and installs it directly to `~/.local/bin/cassette`.

*(Ensure `~/.local/bin` is in your `$PATH`)*.

---

## Setup (5 Minutes)

Spotify requires a free Client ID to talk to their API with PKCE authorization.

### 1. Get a Free Spotify Client ID
1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).
2. Log in and click **Create App**.
3. Fill in:
   - **App name**: `cassette`
   - **App description**: `Terminal cassette player`
   - **Redirect URI**: `http://127.0.0.1:8080/callback` (and `http://127.0.0.1:5588/login` for librespot)
   - Check **Web API** and accept terms.
4. Open the app settings and copy your **Client ID**.

### 2. Configure Cassette
Create `~/.config/cassette/config.yml`:

```yaml
auth:
  client_id: <YOUR_SPOTIFY_CLIENT_ID>
```

*(Replace `<YOUR_SPOTIFY_CLIENT_ID>` with your actual client ID)*.

### 3. One-Time Audio Setup (Optional)
To authorize your terminal as a local audio speaker:

```bash
cassette setup
```

A browser window will pop up asking you to approve Spotify Connect. Once approved, `cassette` will permanently register as your PC's playback device.

---

## Usage

Simply run:

```bash
cassette
```

### Keybindings

| Key | Action |
|---|---|
| `Space` | Play / Pause |
| `n` | Next Track |
| `p` | Previous Track |
| `+` / `-` | Volume Up / Down |
| `>` / `<` | Seek Forward / Backward (5s) |
| `s` | Toggle Shuffle |
| `/` | Search Tracks, Artists, Playlists |
| `Tab` / `P` | Toggle Library / Media Panel |
| `d` | Switch Playback Device |
| `?` | Toggle Help Menu |
| `Ctrl+C` | Quit |

---

## Keywords

`spotify` • `tui` • `terminal` • `cassette` • `retro` • `ansi-art` • `ascii-art` • `librespot` • `music-player` • `cli` • `bubbletea` • `lipgloss` • `golang` • `linux`

---

## License

MIT © [OreoMuncher45](https://github.com/OreoMuncher45)
