package app

import (
	"testing"

	"cassette/core/playlist"
	"cassette/core/ytmusic"
)

func TestYouTubeQueueNavigation(t *testing.T) {
	model := NewModel()
	tracks := []ytmusic.Track{
		{VideoID: "1", Title: "Track 1", Artist: "Artist 1", DurationMs: 180000},
		{VideoID: "2", Title: "Track 2", Artist: "Artist 2", DurationMs: 200000},
		{VideoID: "3", Title: "Track 3", Artist: "Artist 3", DurationMs: 220000},
	}
	model.ytQueue = tracks
	model.ytQueueIndex = 0

	// Advance to next track
	cmd := model.playNextYtTrackCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd on playNextYtTrackCmd")
	}
	if model.ytQueueIndex != 1 {
		t.Fatalf("expected ytQueueIndex to be 1, got %d", model.ytQueueIndex)
	}
	if model.songInfo.Title != "Track 2" {
		t.Fatalf("expected title to be 'Track 2', got %q", model.songInfo.Title)
	}

	// Advance to next track again
	cmd = model.playNextYtTrackCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd on playNextYtTrackCmd")
	}
	if model.ytQueueIndex != 2 {
		t.Fatalf("expected ytQueueIndex to be 2, got %d", model.ytQueueIndex)
	}
	if model.songInfo.Title != "Track 3" {
		t.Fatalf("expected title to be 'Track 3', got %q", model.songInfo.Title)
	}

	// At end of queue, should return nil
	cmd = model.playNextYtTrackCmd()
	if cmd != nil {
		t.Fatal("expected nil cmd at end of queue")
	}
	if model.ytQueueIndex != 2 {
		t.Fatalf("expected ytQueueIndex to stay 2, got %d", model.ytQueueIndex)
	}

	// Previous track
	model.songInfo.Position = 1000 // < 3000ms
	cmd = model.playPrevYtTrackCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd on playPrevYtTrackCmd")
	}
	if model.ytQueueIndex != 1 {
		t.Fatalf("expected ytQueueIndex to be 1, got %d", model.ytQueueIndex)
	}
	if model.songInfo.Title != "Track 2" {
		t.Fatalf("expected title to be 'Track 2', got %q", model.songInfo.Title)
	}
}

func TestLocalPlaylistIntegration(t *testing.T) {
	pl, err := playlist.CreatePlaylist("Test Integration")
	if err != nil {
		t.Fatalf("failed to create playlist: %v", err)
	}
	defer func() {
		_ = playlist.DeletePlaylist(pl.ID)
	}()

	track := ytmusic.Track{
		VideoID:    "xyz123",
		Title:      "Discovery",
		Artist:     "Daft Punk",
		DurationMs: 240000,
	}
	if err := playlist.AddTrackToPlaylist(pl.ID, track); err != nil {
		t.Fatalf("failed to add track: %v", err)
	}

	fetched, err := playlist.GetPlaylist(pl.ID)
	if err != nil {
		t.Fatalf("failed to fetch playlist: %v", err)
	}
	if len(fetched.Tracks) != 1 || fetched.Tracks[0].Title != "Discovery" {
		t.Fatalf("unexpected tracks: %+v", fetched.Tracks)
	}
}
