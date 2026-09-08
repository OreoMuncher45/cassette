// Package spotify adapts the daemon's browsing API to the UI's data types.
package spotify

import (
	"context"
	"fmt"
	"github.com/dubeyKartikay/lazyspotify/librespot"
	spotifyapi "github.com/zmb3/spotify/v2"
	"net/url"
	"strconv"
	"strings"
)

type SpotifyClient struct{ client *librespot.LibrespotApiClient }

func NewSpotifyClient(client *librespot.LibrespotApiClient) *SpotifyClient {
	return &SpotifyClient{client: client}
}
func browsePage[T any](ctx context.Context, s *SpotifyClient, kind string, query url.Values) (*T, error) {
	var page T
	if err := s.client.Browse(ctx, kind, query, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
func pageQuery(offset, limit int) url.Values {
	return url.Values{"offset": {strconv.Itoa(offset)}, "limit": {strconv.Itoa(limit)}}
}
func (s *SpotifyClient) GetUserID(ctx context.Context) (string, error) {
	user, err := browsePage[struct {
		ID string `json:"id"`
	}](ctx, s, "user", nil)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}
func (s *SpotifyClient) GetFirstSavedTrack(ctx context.Context) (string, error) {
	track, err := browsePage[struct {
		URI string `json:"uri"`
	}](ctx, s, "first-track", nil)
	if err != nil {
		return "", err
	}
	return track.URI, nil
}
func (s *SpotifyClient) GetFollowedArtists(ctx context.Context, after string) (*spotifyapi.FullArtistCursorPage, error) {
	// Pathfinder library pages use an offset cursor instead of an artist ID.
	if after == "" {
		after = "0"
	}
	return browsePage[spotifyapi.FullArtistCursorPage](ctx, s, "artists", url.Values{"offset": {after}, "limit": {"10"}})
}
func (s *SpotifyClient) GetPlaylistTracks(ctx context.Context, uri string, offset int) ([]spotifyapi.FullTrack, error) {
	if _, err := idFromURI(uri); err != nil {
		return nil, err
	}
	page, err := s.client.ResolvePlaylistTracks(ctx, uri, offset, 10)
	if err != nil {
		return nil, err
	}
	tracks := make([]spotifyapi.FullTrack, 0, len(page.Tracks))
	for _, item := range page.Tracks {
		var track spotifyapi.FullTrack
		track.URI, track.Name, track.Duration = spotifyapi.URI(item.URI), item.Name, spotifyapi.Numeric(item.DurationMs)
		for _, name := range item.Artists {
			track.Artists = append(track.Artists, spotifyapi.SimpleArtist{Name: name})
		}
		track.Album.Name, track.Album.URI = item.AlbumName, spotifyapi.URI(item.AlbumURI)
		track.Album.Images = []spotifyapi.Image{{URL: item.Img}}
		tracks = append(tracks, track)
	}
	return tracks, nil
}
func idFromURI(uri string) (string, error) {
	parts := strings.Split(uri, ":")
	if len(parts) < 3 || parts[0] != "spotify" || parts[len(parts)-1] == "" {
		return "", fmt.Errorf("invalid spotify uri: %s", uri)
	}
	return parts[len(parts)-1], nil
}
func (s *SpotifyClient) GetUserPlaylists(ctx context.Context, offset int) (*spotifyapi.SimplePlaylistPage, error) {
	q := pageQuery(offset, 10)

	return browsePage[spotifyapi.SimplePlaylistPage](ctx, s, "playlists", q)
}
func (s *SpotifyClient) GetSavedTracks(ctx context.Context, offset int) (*spotifyapi.SavedTrackPage, error) {
	q := pageQuery(offset, 10)

	return browsePage[spotifyapi.SavedTrackPage](ctx, s, "tracks", q)
}
func (s *SpotifyClient) GetSavedAlbums(ctx context.Context, offset int) (*spotifyapi.SavedAlbumPage, error) {
	q := pageQuery(offset, 10)

	return browsePage[spotifyapi.SavedAlbumPage](ctx, s, "albums", q)
}
func (s *SpotifyClient) GetArtistAlbums(ctx context.Context, uri string, offset int) (*spotifyapi.SimpleAlbumPage, error) {
	q := pageQuery(offset, 10)
	q.Set("uri", uri)
	return browsePage[spotifyapi.SimpleAlbumPage](ctx, s, "artist-albums", q)
}
func (s *SpotifyClient) GetAlbumTracks(ctx context.Context, uri string, offset int) (*spotifyapi.SimpleTrackPage, error) {
	q := pageQuery(offset, 50)
	q.Set("uri", uri)
	return browsePage[spotifyapi.SimpleTrackPage](ctx, s, "album-tracks", q)
}
func (s *SpotifyClient) SearchPlaylists(ctx context.Context, query string, offset, limit int) (*spotifyapi.SimplePlaylistPage, error) {
	q := pageQuery(offset, limit)
	q.Set("q", query)
	return browsePage[spotifyapi.SimplePlaylistPage](ctx, s, "search-playlists", q)
}
func (s *SpotifyClient) SearchTracks(ctx context.Context, query string, offset, limit int) (*spotifyapi.FullTrackPage, error) {
	q := pageQuery(offset, limit)
	q.Set("q", query)
	return browsePage[spotifyapi.FullTrackPage](ctx, s, "search-tracks", q)
}
func (s *SpotifyClient) SearchAlbums(ctx context.Context, query string, offset, limit int) (*spotifyapi.SimpleAlbumPage, error) {
	q := pageQuery(offset, limit)
	q.Set("q", query)
	return browsePage[spotifyapi.SimpleAlbumPage](ctx, s, "search-albums", q)
}
func (s *SpotifyClient) SearchArtists(ctx context.Context, query string, offset, limit int) (*spotifyapi.FullArtistPage, error) {
	q := pageQuery(offset, limit)
	q.Set("q", query)
	return browsePage[spotifyapi.FullArtistPage](ctx, s, "search-artists", q)
}
