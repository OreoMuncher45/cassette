package source

// SourceKind identifies the active music backend.
type SourceKind string

const (
	SourceYouTubeMusic SourceKind = "ytmusic"
	SourceSpotify      SourceKind = "spotify"
)

// TrackInfo is a backend-agnostic track representation used throughout the UI.
type TrackInfo struct {
	ID         string
	Title      string
	Artist     string
	Album      string
	DurationMs int
	ArtURL     string
	StreamURI  string // Backend-specific URI (spotify:track:xxx or youtube video ID)
}

// SearchResults holds unified search results from any backend.
type SearchResults struct {
	Tracks  []TrackInfo
	Albums  []AlbumInfo
	Artists []ArtistInfo
}

// AlbumInfo is a backend-agnostic album representation.
type AlbumInfo struct {
	ID     string
	Title  string
	Artist string
	ArtURL string
	Tracks []TrackInfo
}

// ArtistInfo is a backend-agnostic artist representation.
type ArtistInfo struct {
	ID     string
	Name   string
	ArtURL string
}

// PlaylistInfo is a backend-agnostic playlist representation.
type PlaylistInfo struct {
	ID     string
	Title  string
	Owner  string
	ArtURL string
	Tracks []TrackInfo
}

// MusicSource is the unified interface that both YouTube Music and Spotify backends implement.
type MusicSource interface {
	// Kind returns the backend identifier.
	Kind() SourceKind

	// Search performs a text search and returns unified results.
	Search(query string, limit int) (*SearchResults, error)

	// GetAlbumTracks returns all tracks in an album.
	GetAlbumTracks(albumID string) ([]TrackInfo, error)

	// GetArtistAlbums returns albums by an artist.
	GetArtistAlbums(artistID string) ([]AlbumInfo, error)

	// GetPlaylistTracks returns all tracks in a playlist.
	GetPlaylistTracks(playlistID string) ([]TrackInfo, error)

	// GetStreamURL resolves a track to a playable audio URL or local stream address.
	GetStreamURL(track TrackInfo) (string, error)

	// SeedRadio queues similar tracks based on a seed track for endless playback.
	SeedRadio(seedTrack TrackInfo, count int) ([]TrackInfo, error)
}

// Playback controls the audio output (mpv for YouTube Music, librespot/Spotify Connect for Spotify).
type Playback interface {
	// Play starts playback of the given stream URL.
	Play(streamURL string) error

	// Pause toggles pause.
	Pause() error

	// Resume resumes playback.
	Resume() error

	// Stop stops playback entirely.
	Stop() error

	// Seek seeks to the given position in milliseconds.
	Seek(positionMs int) error

	// SeekRelative seeks relative to the current position.
	SeekRelative(offsetMs int) error

	// SetVolume sets volume (0-100).
	SetVolume(percent int) error

	// GetVolume returns current volume (0-100).
	GetVolume() int

	// PositionMs returns the current playback position in milliseconds.
	PositionMs() int

	// DurationMs returns the duration of the current track in milliseconds.
	DurationMs() int

	// IsPlaying returns true if audio is currently playing.
	IsPlaying() bool
}
