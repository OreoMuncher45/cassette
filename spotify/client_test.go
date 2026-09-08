package spotify

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/dubeyKartikay/lazyspotify/librespot"
)

func newTestClient(t *testing.T, handler http.Handler) *SpotifyClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	host, portString, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portString)
	return NewSpotifyClient(librespot.NewLibrespotApiClient(librespot.NewLibrespotApiServer(host, port)))
}

func TestEveryLookupUsesDaemon(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name, path, response string
		call                 func(*SpotifyClient) error
	}{
		{"user", "/browse/user", `{"id":"alice"}`, func(c *SpotifyClient) error { _, e := c.GetUserID(ctx); return e }},
		{"first", "/browse/first-track", `{"uri":"spotify:track:first"}`, func(c *SpotifyClient) error { _, e := c.GetFirstSavedTrack(ctx); return e }},
		{"playlists", "/browse/playlists", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.GetUserPlaylists(ctx, 20); return e }},
		{"tracks", "/browse/tracks", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.GetSavedTracks(ctx, 20); return e }},
		{"albums", "/browse/albums", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.GetSavedAlbums(ctx, 20); return e }},
		{"artists", "/browse/artists", `{"items":[],"total":0,"cursors":{"after":""}}`, func(c *SpotifyClient) error { _, e := c.GetFollowedArtists(ctx, "20"); return e }},
		{"artist albums", "/browse/artist-albums", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.GetArtistAlbums(ctx, "spotify:artist:abc", 20); return e }},
		{"album tracks", "/browse/album-tracks", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.GetAlbumTracks(ctx, "spotify:album:abc", 20); return e }},
		{"search playlists", "/browse/search-playlists", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.SearchPlaylists(ctx, "blue & train", 20, 10); return e }},
		{"search tracks", "/browse/search-tracks", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.SearchTracks(ctx, "blue & train", 20, 10); return e }},
		{"search albums", "/browse/search-albums", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.SearchAlbums(ctx, "blue & train", 20, 10); return e }},
		{"search artists", "/browse/search-artists", `{"items":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.SearchArtists(ctx, "blue & train", 20, 10); return e }},
		{"playlist tracks", "/resolver/tracks", `{"uri":"spotify:playlist:abc","tracks":[],"total":0}`, func(c *SpotifyClient) error { _, e := c.GetPlaylistTracks(ctx, "spotify:playlist:abc", 20); return e }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != tt.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if tt.name != "user" && tt.name != "first" && r.URL.Query().Get("offset") != "20" {
					t.Error("lost offset")
				}
				if len(tt.name) >= 6 && tt.name[:6] == "search" && r.URL.Query().Get("q") != "blue & train" {
					t.Error("lost search query")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			if err := tt.call(c); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSavedTracksAndArtistCursorDecode(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/browse/artists" {
			_, _ = w.Write([]byte(`{"items":[{"name":"Artist","uri":"spotify:artist:a"}],"total":21,"cursors":{"after":"10"}}`))
		} else {
			_, _ = w.Write([]byte(`{"items":[{"track":{"name":"Song","uri":"spotify:track:t","artists":[{"name":"Artist"}],"album":{"name":"Album","images":[{"url":"cover"}]}}}],"total":1}`))
		}
	}))
	tracks, err := c.GetSavedTracks(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks.Tracks) != 1 || tracks.Tracks[0].Name != "Song" || tracks.Tracks[0].Album.Images[0].URL != "cover" {
		t.Fatalf("tracks = %+v", tracks)
	}
	artists, err := c.GetFollowedArtists(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if artists.Cursor.After != "10" || len(artists.Artists) != 1 {
		t.Fatalf("artists = %+v", artists)
	}
}

func TestBrowseRejectsUnpairedOldAndMalformedDaemons(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"unpaired", 204, ""}, {"old binary", 200, `{"playback_ready":true}`},
		{"forbidden", 403, ""}, {"malformed", 200, "{"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			if _, err := c.GetSavedTracks(context.Background(), 0); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
