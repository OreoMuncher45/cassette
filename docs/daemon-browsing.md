# Daemon browsing migration

All methods in `spotify/client.go` now call the local daemon. The Spotify Go
package is retained for response types used by the UI, not for network requests.
Startup uses device authorization and waits for daemon readiness before loading
the library. The UI polls `GET /auth/code` and shows the pairing URL, code, and
expiry. Existing OAuth settings and Keychain tokens are not used.

| Client operation | Daemon endpoint |
| --- | --- |
| User ID | GET /browse/user |
| First saved track | GET /browse/first-track |
| User playlists | GET /browse/playlists |
| Saved tracks | GET /browse/tracks |
| Saved albums | GET /browse/albums |
| Followed artists | GET /browse/artists |
| Artist albums | GET /browse/artist-albums?uri=spotify:artist:… |
| Album tracks | GET /browse/album-tracks?uri=spotify:album:… |
| Playlist tracks | GET /resolver/tracks?uri=spotify:playlist:… |
| Search playlists/tracks/albums/artists | GET /browse/search-{kind}?q=… |

Browse pages accept `offset` and `limit` (1–50). Artists now use a numeric
offset cursor; it is returned in `cursors.after`. The daemon owns Spotify's
private request formats and converts them to the existing page shapes.
The frontend checks response fields to detect an older daemon that returns
its root health response for an unknown route.

## Build both repositories

From lazyspotify, with the modified fork alongside it:

```sh
./scripts/release/build-librespot-daemon.sh \
  --source-dir ../go-librespot-lazyspotify \
  --output ../go-librespot-lazyspotify/dist/lazyspotify-librespot
go build -o /tmp/lazyspotify-browse ./cmd/lazyspotify
```

Set `librespot.daemon.cmd` in the app config to the absolute path of that
daemon, then launch the new app and approve the displayed device authorization code.
The fork changes are in a separate checkout and must be released before
updating the release manifest. The existing manifest still pins the previous
published daemon; no release has been created by this migration.

## Verification

Run `go test ./...` in lazyspotify and go-librespot. Unit tests exercise every
client route, private request construction, response mapping, cursor handling,
validation, and error responses. They do not log in to Spotify.

Private Pathfinder operations use persisted query hashes, which Spotify can
change. GraphQL errors and missing fields are reported instead of being treated
as an empty library. Live browsing must also be checked after pairing; passing
unit tests does not establish that Spotify accepts the session or query hashes.
