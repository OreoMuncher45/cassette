package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"cassette/ui/v1/common"
	spotifyapi "github.com/zmb3/spotify/v2"
)

func TestCursorEncodingAndDecodingAreDefensive(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
		want   int
	}{
		{name: "empty", cursor: "", want: 0},
		{name: "valid", cursor: "20", want: 20},
		{name: "negative", cursor: "-10", want: 0},
		{name: "invalid", cursor: "not-a-number", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeOffsetCursor(tt.cursor); got != tt.want {
				t.Fatalf("decodeOffsetCursor(%q) = %d, want %d", tt.cursor, got, tt.want)
			}
		})
	}

	if got := encodeOffsetCursor(-20); got != "0" {
		t.Fatalf("encodeOffsetCursor(-20) = %q, want 0", got)
	}
	if got := encodeOffsetCursor(30); got != "30" {
		t.Fatalf("encodeOffsetCursor(30) = %q, want 30", got)
	}
}

func TestPaginationFromOffsetReflectsCurrentPageAndNextCursor(t *testing.T) {
	got := paginationFromOffset(10, 10, 25, 10)
	want := common.PaginationInfo{
		CurrentPage: 2,
		TotalPages:  3,
		TotalItems:  25,
		HasNext:     true,
		NextCursor:  "20",
	}
	if got != want {
		t.Fatalf("paginationFromOffset() = %#v, want %#v", got, want)
	}

	got = paginationFromOffset(20, 5, 25, 10)
	want = common.PaginationInfo{
		CurrentPage: 3,
		TotalPages:  3,
		TotalItems:  25,
		HasNext:     false,
		NextCursor:  "",
	}
	if got != want {
		t.Fatalf("paginationFromOffset(last page) = %#v, want %#v", got, want)
	}
}

func TestHandleMediaRequestDefaultsPageBeforeDispatch(t *testing.T) {
	model := NewModel()

	var gotRequest common.MediaRequest
	model.requestHandlers[common.GetSavedTracks] = func(request common.MediaRequest) tea.Cmd {
		gotRequest = request
		return func() tea.Msg {
			return "handled"
		}
	}

	cmd := model.handleMediaRequest(common.MediaRequest{Kind: common.GetSavedTracks})
	if cmd == nil {
		t.Fatal("handleMediaRequest() = nil, want dispatch command")
	}
	if msg := cmd(); msg != "handled" {
		t.Fatalf("handler message = %#v, want handled", msg)
	}
	if gotRequest.Page != 1 {
		t.Fatalf("dispatched Page = %d, want 1", gotRequest.Page)
	}
}

func TestHandleMediaRequestReturnsNilForUnknownKind(t *testing.T) {
	model := NewModel()

	cmd := model.handleMediaRequest(common.MediaRequest{Kind: common.MediaRequestKind(999), Page: 1})
	if cmd != nil {
		t.Fatalf("handleMediaRequest(unknown) = %#v, want nil", cmd)
	}
}

func TestAdaptSpotifySavedTracksBuildsUserFacingEntity(t *testing.T) {
	page := &spotifyapi.SavedTrackPage{
		Tracks: []spotifyapi.SavedTrack{
			{
				FullTrack: spotifyapi.FullTrack{
					SimpleTrack: spotifyapi.SimpleTrack{
						Name: "Track",
						URI:  "spotify:track:track-id",
						Artists: []spotifyapi.SimpleArtist{
							{Name: "Artist One"},
							{Name: "Artist Two"},
						},
					},
					Album: spotifyapi.SimpleAlbum{
						Name:   "Album",
						Images: []spotifyapi.Image{{URL: "https://images.spotify.test/large.jpg"}, {URL: "https://images.spotify.test/small.jpg"}},
					},
				},
			},
		},
	}

	entities := adaptSpotifySavedTracks(page)
	if len(entities) != 1 {
		t.Fatalf("len(entities) = %d, want 1", len(entities))
	}

	want := common.NewEntity("Track", "Artist One, Artist Two • Album", "spotify:track:track-id", "https://images.spotify.test/large.jpg")
	if entities[0] != want {
		t.Fatalf("entity = %#v, want %#v", entities[0], want)
	}
}

func TestAdaptSpotifyTracksBuildsUserFacingEntity(t *testing.T) {
	tracks := []spotifyapi.FullTrack{
		{
			SimpleTrack: spotifyapi.SimpleTrack{
				Name: "Track One",
				URI:  "spotify:track:one",
				Artists: []spotifyapi.SimpleArtist{
					{Name: "Artist A"},
				},
			},
			Album: spotifyapi.SimpleAlbum{
				Name:   "Album X",
				Images: []spotifyapi.Image{{URL: "https://images.spotify.test/one.jpg"}},
			},
		},
	}

	entities := adaptSpotifyTracks(tracks)
	if len(entities) != 1 {
		t.Fatalf("len(entities) = %d, want 1", len(entities))
	}
	want := common.NewEntity("Track One", "Artist A • Album X", "spotify:track:one", "https://images.spotify.test/one.jpg")
	if entities[0] != want {
		t.Fatalf("entity = %#v, want %#v", entities[0], want)
	}
}

func TestPlayerStateMsgUpdatesPlaybackState(t *testing.T) {
	model := NewModel()
	model.playerReady = true

	msg := playerStateMsg{
		state: &spotifyapi.PlayerState{
			CurrentlyPlaying: spotifyapi.CurrentlyPlaying{
				Playing:  true,
				Progress: 42000,
				Item: &spotifyapi.FullTrack{
					SimpleTrack: spotifyapi.SimpleTrack{
						Name: "Test Track",
						Artists: []spotifyapi.SimpleArtist{
							{Name: "Test Artist"},
						},
						Duration: 180000,
					},
					Album: spotifyapi.SimpleAlbum{Name: "Test Album"},
				},
			},
			Device: spotifyapi.PlayerDevice{
				ID:     "dev-1",
				Name:   "Speaker",
				Volume: 80,
			},
			ShuffleState: true,
		},
	}

	model.Update(msg)

	if !model.playing {
		t.Fatal("playing = false, want true")
	}
	if model.songInfo.Title != "Test Track" {
		t.Fatalf("song title = %q, want Test Track", model.songInfo.Title)
	}
	if model.songInfo.Position != 42000 {
		t.Fatalf("song position = %d, want 42000", model.songInfo.Position)
	}
	if model.volumeInfo.Volume != 80 {
		t.Fatalf("volume = %d, want 80", model.volumeInfo.Volume)
	}
}
