<div align="center">

```
╭░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░╮
│ (o)  ╭───╮                                                         (o) │
│      │ B │  ──────────────────────────────────────────  [ ]IN [ ]OUT   │
│      ╰───╯                                                             │
│  ╭──────────────────────────────────────────────────────────────────╮  │
│  │               CASSETTE • YOUTUBE MUSIC & SPOTIFY TUI              │  │
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

**A high-fidelity, retro ANSI animated cassette tape music player — YouTube Music first, Spotify supported.**  
*Because every other music client is either a 500MB Electron dumpster fire or looks like an Excel spreadsheet from 1994.*

[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux-blue.svg)](https://github.com)
[![Status](https://img.shields.io/badge/status-actively%20developed-brightgreen.svg)](#roadmap)

**Actively developed** — see the [Roadmap](#roadmap) for what's shipping next.

</div>

# Preview
<a href="https://imgbb.com/"><img src="https://i.ibb.co/5xcF3JbB/32320.gif" alt="32320" border="0"></a>
---

## Why does this exist? (The Rant)

Because modern Spotify clients are fucking awful.

Somehow, we managed to take **playing music** and turn it into an exercise in tolerating bloated software, shitty UX, and a dozen layers of bullshit that nobody asked for.

1. **The Official Spotify Desktop App**

   This thing is an absolute resource hog. Gigabytes of RAM to play fucking music. Web views inside web views inside whatever other Electron-flavored nightmare they've stuffed in there. Podcasts shoved in your face. Recommendations you didn't ask for. Clutter everywhere.

   And somehow, despite all that horsepower, you can still sit there waiting for the goddamn search bar to become usable.

2. **Generic "Terminal" Spotify TUIs**

   Okay, so let's escape the bloated desktop app and use a terminal client.

   Except half of them look like they were designed by someone whose idea of a UI is a white table dumped straight out of a database.

   Lifeless rows. Lifeless borders. Lifeless text.

   They technically work. Great. So does a fucking spreadsheet.

   Music isn't supposed to feel like filing your taxes.

3. **The "Play on Device" Fuckery**

   And then there's the other brilliant idea: terminal clients that don't actually fucking play music.

   You hit play.

   Nothing.

   Oh, right. You need to configure some external daemon, connect a D-Bus socket, install another audio server, sacrifice a goat to PulseAudio, and spend the next four hours figuring out why the goddamn thing still isn't making sound.

   **No.**

   I'm not doing that.

**Enough.**

I built `cassette` out of pure spite because I wanted a music player that didn't suck.

The idea is stupidly simple:

**Music should feel like physical media again.**

So instead of another sterile terminal table, `cassette` gives you a handcrafted, pixel-aligned ANSI cassette tape that **actually spins while your music is playing**.

The reels move. The tape transfers between the supply and take-up spools. The metadata sits in the HUD. The whole fucking thing lives inside your terminal.

And, crucially, **it actually plays the music.**

`cassette` connects to **YouTube Music** by default (free, no account needed) using `mpv` + `yt-dlp`, or optionally to **Spotify** via `librespot`. Zero API keys for YouTube Music. Zero Premium account needed.

Search for a track, press play, and you're listening.

That's it.

No Electron bloat.

No podcast bullshit.

No "please configure this external service before sound will come out of your speakers."

No dead-ass terminal UI pretending that functionality is enough.

Just your music, your keyboard, and a fucking cassette tape spinning in your terminal.

---

## Features

* **YouTube Music First**
  Free streaming with zero credentials or API keys. Studio version preference (Audio Track Videos) ensures you hear the official studio master, not a music video rip with extra sound effects.

* **Dual-Source Engine**
  Toggle between YouTube Music (free) and Spotify (Premium) anytime. Both backends are fully supported.

* **Animated ANSI Cassette Art**
  Handcrafted, pixel-aligned cassette animation with realistic rotating 6-tooth gear sprockets, calibrated tape window, dynamic supply/take-up spools, real-time tape transfer, and an integrated metadata HUD.

* **Word-Synced Karaoke Lyrics**
  Multi-source word-by-word lyrics highlighting from Apple Music (TTML), QQ Music, and LRCLIB. Per-word timing when available, line-synced fallback otherwise.

* **Actually Plays Music**
  YouTube Music via `mpv` + `yt-dlp`, or Spotify via bundled `librespot` Connect. Your computer appears directly as `cassette` in Spotify. No external audio daemon circus.

* **Endless Song Radio**
  Start playing any track from search and `cassette` automatically seeds similar-track recommendations, continuously building the queue so you can just let the fucking music run.

* **Zero-CPU Background Mode**
  When the terminal loses focus or is minimized, cassette stops all ANSI animation rendering and album art rasterization. CPU drops to ~0%. Resumes instantly on re-focus.

* **Distraction-Free Keyboard Controls**
  Search, browse playlists, tracks and albums, switch devices, control volume, seek, shuffle, and manage playback without touching a mouse.

* **Zero Electron Bloat**
  Written in pure Go with Bubble Tea and Lipgloss. Fast startup, low CPU usage, minimal memory footprint, and none of the bullshit that comes with shipping an entire browser just to play a song.

---

## Requirements

### YouTube Music Mode (Default — Free)
- **Linux** (PulseAudio / PipeWire).
- **mpv** (audio playback engine).
- **yt-dlp** (YouTube audio extraction).
- Standard terminal with UTF-8 support (Kitty, Konsole, Alacritty, WezTerm, etc.).

### Spotify Mode (Optional)
- **Spotify Premium** account.
- **librespot** (Spotify Connect audio daemon).
- A free Spotify Client ID (see setup below).

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

## Setup

Run the setup assistant:

```bash
cassette setup
```

You'll be prompted to choose a profile:

* **[1] YouTube Music (Default — Recommended)**
  - Free streaming, no account or API key needed.
  - Installs `mpv` and `yt-dlp` for audio playback and YouTube audio extraction.
  - Studio version preference: plays Audio Track Videos (studio masters) over music videos.
  - Direct flag: `cassette setup --ytmusic`

* **[2] Spotify (Spotify Connect & Web API)**
  - Requires a [free Spotify Client ID](https://developer.spotify.com/dashboard) + Spotify Premium.
  - Configures `librespot` for native terminal Spotify Connect playback.
  - Direct flag: `cassette setup --spotify`

* **[3] Dual Setup (Both YouTube Music & Spotify)**
  - Installs both backends. Toggle between sources with `F3` in the player.
  - Direct flag: `cassette setup --dual`

* **[4] [✔] Auto-start Cassette on boot**
  - Launches Cassette automatically in your terminal on desktop login.
  - Generates standard FreeDesktop `~/.config/autostart/cassette.desktop`.
  - Toggle flags: `cassette setup --autostart` or `cassette setup --no-autostart`.

### Spotify-Specific Setup

If you chose Spotify mode, you'll also need:

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).
2. Create an app with redirect URI `http://127.0.0.1:8080/callback`.
3. Copy your Client ID to `~/.config/cassette/config.yml`:

```yaml
default_source: spotify
auth:
  client_id: <YOUR_SPOTIFY_CLIENT_ID>
```

A browser window will pop up during setup to authorize Spotify Connect.

---

## Linux Media Controls & Bluetooth Earbuds (MPRIS D-Bus)

Cassette automatically registers on the Linux D-Bus session bus as `org.mpris.MediaPlayer2.cassette`:
- **Bluetooth Earbuds (AVRCP)**: Single-tap play/pause, double-tap next track, and previous track controls work directly from your wireless earbuds.
- **Hardware Keyboard Media Keys**: Physical `XF86AudioPlay`, `XF86AudioPause`, `XF86AudioNext`, and `XF86AudioPrev` keys control Cassette from any workspace or application, even when minimized or when the screen is locked.
- **Desktop & Lock Screen Integration**: Real-time track metadata (title, artist, album, album art, length, position) and playback status are displayed in KDE Plasma media widgets, GNOME media center, lock screens, and command-line tools (`playerctl`).

---

## Usage

Simply run:

```bash
cassette
```

### Keybindings & Controls

The bottom bar provides quick access to both manager interfaces: `F1 - Keybinds  •  F2 - Settings`.

| Key | Action | Description |
|---|---|---|
| **`F1`** | **Keybindings Manager** | Instant searchable keybinds manager with live rebinding (`Enter` to rebind, `Ctrl+R` to reset defaults) |
| **`F2`** | **Visuals & Settings** | Dynamic album art color extraction, RGB effects, themes, animation speeds, and autostart on boot |
| `Space` | Play / Pause | Toggle playback on active device |
| `s` | Toggle Shuffle | Turn shuffle on or off (displays `Shuffle: ON 🔀` / `OFF` HUD toast) |
| `n` | Next Track | Skip to next track in queue |
| `p` | Previous Track | Restart song or go to previous track |
| `+` / `-` | Volume Up / Down | Increase or decrease volume by 5% |
| `>` / `<` | Seek Forward / Back | Jump forward or backward by 5 seconds |
| `/` | Search Library | Search tracks, artists, albums, and playlists |
| `Tab` | Cycle Search Tabs | In search/library, switch between `PL` (Playlists) → `TR` (Tracks) → `AL` (Albums) → `AR` (Artists) |
| `Shift+Tab` | Reverse Cycle Tabs | Cycle search tabs in reverse order |
| `Left` / `Del` | Navigate Back / Queue | Return to parent list when drilled down, or return focus to Queue at root level |
| `Enter` | Select / Play | In `PL`, plays the full playlist context; in `TR`, plays the track & starts Endless Song Radio |
| `l` | Toggle Lyrics | Open / close real-time synced lyrics panel |
| `a` | Toggle Big Artwork | Open / close 24-bit Truecolor terminal album art panel |
| `q` | Toggle Queue | Open / close playback queue panel |
| `d` | Switch Playback Device | Open Spotify Connect device selector |
| `z` | Zen Mode | Focus mode: hides side panels, showing only the spinning cassette deck |
| `Ctrl+C` | Quit | Cleanly exit Cassette |

---

### Visual Settings & Themes (`F2`)

Press **`F2`** at any time to open the visual customization engine:

* **11+ Color Schemes**:
  * **Album Art Reactive**: Automatically samples the currently playing song's album art in real-time, extracting the most vibrant accent palette.
  * **Retro Cassette Cyan**: Classic Hi-Fi turquoise and warm amber HUD.
  * **Cyberpunk Neon**: High-contrast hot neon pink and electric cyan.
  * **Synthwave 80s**: Retro sunset purple and laser neon orange.
  * **Matrix Phosphor**: Green CRT hacker terminal phosphor glow.
  * **Dracula Dark**: Vampire purple, radiant pink, and sky cyan.
  * **Nordic Frost**: Arctic ice blue and calm polar slate.
  * **Monochrome Amber**: Vintage warm amber cassette deck display.
  * **Tokyo Night**: Deep indigo midnight and pastel magenta.
  * **Solarized Dark**: Balanced teal cyan and solar amber gold.
  * **Pastel Dream**: Soft pastel pink and dreamy sky blue.

* **Live Visual Effects**:
  * **Breathing Glow**: Smooth sine-wave brightness pulsing across spools, HUD borders, and accents.
  * **Rainbow RGB Cycle**: Dynamic 360° chromatic wave cycling through the full color spectrum.
  * **Heartbeat Pulse**: Rhythmic bass pulse synced to playback.
  * **Static Colors**: Pure, distraction-free solid theme styling.

* **RGB & Animation Speeds**:
  * **Chill (Slow)** • **Groove (Medium)** • **Hyper (Fast)** • **Ultra (Ludicrous)**

All settings are automatically saved and persistent across sessions.

---

## Keywords

`youtube-music` • `spotify` • `tui` • `terminal` • `cassette` • `retro` • `ansi-art` • `ascii-art` • `librespot` • `mpv` • `yt-dlp` • `music-player` • `cli` • `bubbletea` • `lipgloss` • `golang` • `linux` • `karaoke-lyrics`

---

## Roadmap

cassette is actively developed, and this is what's currently in the pipeline.

### 🚧 In progress — Spotify Soloist API migration

Right now cassette bundles `librespot` for playback. Spotify shipped **[Soloist](https://developer.spotify.com/documentation/soloist)** in August 2026 — an official headless Spotify Connect client for Linux, built on **the same playback engine as the official Spotify apps**, with a local WebSocket API for integrations.

Migrating to it unlocks a stack of things we simply cannot do today:

* **HiFi / lossless audio** — up to 24-bit / 44.1 kHz (Premium).
* **Crossfade & automix** — proper smooth transitions instead of hard cuts (Premium).
* **Loudness normalization** with peak limiting — no more volume jumping between tracks.
* **An official playback engine** — retires a whole class of `librespot`-specific bugs.
* **Jam, Smart Shuffle & Autoplay** — parity with the official Spotify apps.

> Tracking issue: **[#1 — Roadmap: adopt the Spotify Soloist API](https://github.com/OreoMuncher45/cassette/issues/1)**

### Planned

* **Bug squashing pass** — retest the known `librespot`-era failure modes against Soloist and close out whatever it fixes.
* **Event-driven UI** — drive the cassette animation, metadata HUD and queue panel from real-time playback events instead of polling.
* More to come — this list grows.

Got a feature request or a bug to report? **[Open an issue](https://github.com/OreoMuncher45/cassette/issues)** — it genuinely gets read.

---

## Support Cassette

Cassette is free and open source, and it's staying that way. No telemetry, no accounts, no ads, no "Pro" tier, and no plan to add one.

If you enjoy using it and want to throw a few bucks toward development, you can do so here:

| Asset | Address |
|---|---|
| **Nano (XNO)** — instant, feeless, no minimum | `nano_1zqdw3qf1z8k3jx8jintaiwpo3yz7zqh1me4ph5j439ts8hsppx8dzy4xcsz` |
| **USDC on Base** (EVM) | `0x3f262ee685ced4a8270cece45ebdfdb2b18f54b5` |
| **Zcash (ZEC)** — shielded unified address | `u1z9k30yyvy63f5w0jypt02kvvw6dcpcgprlhzmsgc57mw6qc8rtuc5tfd9ny4atqhr448udexhkuc8xgl0z5rd9njuxnwl7kh2ahqwcqlydt9dpr4t40eawr5st74as5jed669993epsnwuejnrwv4yrkx065pqvmt0cr8gdwggcv2djp` |

<details>
<summary><b>Copy-paste block</b></summary>

```
XNO (Nano):
nano_1zqdw3qf1z8k3jx8jintaiwpo3yz7zqh1me4ph5j439ts8hsppx8dzy4xcsz

USDC (Base / EVM):
0x3f262ee685ced4a8270cece45ebdfdb2b18f54b5

ZEC (shielded unified address):
u1z9k30yyvy63f5w0jypt02kvvw6dcpcgprlhzmsgc57mw6qc8rtuc5tfd9ny4atqhr448udexhkuc8xgl0z5rd9njuxnwl7kh2ahqwcqlydt9dpr4t40eawr5st74as5jed669993epsnwuejnrwv4yrkx065pqvmt0cr8gdwggcv2djp
```

</details>

> **Why no GitHub Sponsors button?** Sponsors doesn't pay out in Pakistan, and most conventional payment processors won't either. There's genuinely no button missing by accident — crypto is just what works.

Anything is appreciated, nothing is expected. If you'd rather help without spending money, a star, a bug report, or a pull request is just as welcome.

---

## Credits

* Forked and evolved from [lazyspotify](https://github.com/ccharles13/lazyspotify) by Charles C.

---

## License

MIT © [OreoMuncher45](https://github.com/OreoMuncher45)
