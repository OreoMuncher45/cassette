> ✍️ I write about building tools like this at [dubeykartikay.com](https://dubeykartikay.com).

<div align="center">
  <img width="220" src="./docs/assets/logo.png" alt="lazyspotify logo" />
  <h1>lazyspotify</h1>
  <p>A terminal Spotify client.</p>
  <p>
    <a href="https://github.com/dubeyKartikay/lazyspotify/releases"><img src="https://img.shields.io/github/v/release/dubeyKartikay/lazyspotify?label=release" alt="Release"/></a>
    <a href="https://github.com/dubeyKartikay/lazyspotify/actions/workflows/release.yml"><img src="https://github.com/dubeyKartikay/lazyspotify/actions/workflows/release.yml/badge.svg" alt="Release workflow"/></a>
    <a href="./LICENSE"><img src="https://img.shields.io/github/license/dubeyKartikay/lazyspotify" alt="License"/></a>
  </p>
</div>


![LazySpotify](docs/assets/lazyspotify-start.png)

## Requirements

- A Spotify Premium account.
- The patched `lazyspotify-librespot` daemon (only if you are installing from source or running an unpackaged build)

## Install

### Homebrew

```bash
brew tap dubeyKartikay/lazyspotify
brew install lazyspotify
```

### Arch Linux

```bash
yay -S lazyspotify-bin
```

### Nix

Available in [Nixpkgs](https://search.nixos.org/packages?channel=unstable&query=lazyspotify#show=lazyspotify).

```bash
nix run nixpkgs#lazyspotify
```

### GitHub Releases

Download the latest package from [GitHub Releases](https://github.com/dubeyKartikay/lazyspotify/releases).

- macOS: signed `.zip`
- Ubuntu/Debian: `.deb`
- Fedora/RHEL: `.rpm`
- Arch: `.tar.gz`

Example package installs:

```bash
sudo dpkg -i lazyspotify-*.deb
sudo dnf install ./lazyspotify-*.rpm
```

### Build From Source

Build the app:

```bash
git clone https://github.com/dubeyKartikay/lazyspotify.git
cd lazyspotify
make build
```

Then build the patched daemon from [`dubeyKartikay/go-librespot`](https://github.com/dubeyKartikay/go-librespot) and point `librespot.daemon.cmd` at that binary in your config.

If you build `lazyspotify` yourself and do not compile in a packaged daemon path, `librespot.daemon.cmd` is required.

## Demos

### Play A Track From Playlist > Track

![Play a track from a playlist](docs/assets/demos/playlist-track-play.gif)

### Player Controls

![Player controls](docs/assets/demos/player-controls.gif)

### Library Navigation

![Library navigation](docs/assets/demos/library-navigation.gif)

### Search

![Search navigation](docs/assets/demos/search-navigation.gif)

## Pair with Spotify

Start lazyspotify and open the pairing link shown in the terminal on any phone
or computer. Enter the displayed code if prompted and approve access. Press
`c` to copy the link. The daemon stores credentials for subsequent launches.
No developer client ID, local callback server, or same-network discovery is needed.

Library browsing, search, and metadata requests go through the patched daemon.
This development version requires a daemon with `/browse/` and `/auth/code` endpoints;
the previously released v0.7.1.1 daemon does not include them. Rebuild both
repositories together (see [daemon API migration](docs/daemon-browsing.md)).

## Configuration

Config file locations:

- macOS: `~/Library/Application Support/lazyspotify/config.yml`
- Linux: `~/.config/lazyspotify/config.yml`

If the file is missing, lazyspotify creates a comment-only `config.yml`.
Package installs use defaults. Legacy `auth.*` settings are ignored by startup.

Minimal config for source or manual installs:

```yaml
librespot:
  daemon:
    cmd:
      - /absolute/path/to/lazyspotify-librespot
```

The generated daemon config is written automatically under the `librespot/` subdirectory inside the app config directory. You usually do not need to edit it manually.

### Logging

| Key | Required | Default | Notes |
| --- | --- | --- | --- |
| `log_level` | No | `ERROR` | App log level. |

### Librespot Settings

| Key | Required | Default | Notes |
| --- | --- | --- | --- |
| `librespot.host` | No | `127.0.0.1` | Host for the local playback API server. |
| `librespot.port` | No | `4040` | Port for the local playback API server. |
| `librespot.timeout` | No | `180` | Playback API timeout in seconds. |
| `librespot.retry-delay` | No | `100` | Retry delay in milliseconds. |
| `librespot.max-retries` | No | `3` | Retry count for daemon calls. |
| `librespot.seek-step-ms` | No | `5000` | Seek step size in milliseconds. |
| `librespot.volume-step` | No | `20` | Volume step percentage (0-100) used for volume controls. |
| `librespot.daemon.cmd` | Sometimes | none | Required for source/manual installs unless a packaged daemon path was compiled into the binary. |
| `librespot.daemon.log_level` | No | `ERROR` | Log level written into the generated librespot daemon config. |
Authentication uses `device_auth`. Spotify Connect discovery is disabled by
default; set `librespot.daemon.zeroconf_enabled: true` to keep advertising locally.

Environment variables can override config values by replacing `.` and `-` with `_`. Examples: `LOG_LEVEL`, `LIBRESPOT_DAEMON_LOG_LEVEL`.

## Run

Start the app with:

```bash
lazyspotify
```

If you built from source:

```bash
./target/lazyspotify
```

Print build metadata:

```bash
lazyspotify version
```

## Development

```bash
make run
go test ./...
```

## Community

- [Code of Conduct](./.github/CODE_OF_CONDUCT.md)
- [Contributing Guide](./.github/CONTRIBUTING.md)
- [Security Policy](./.github/SECURITY.md)
