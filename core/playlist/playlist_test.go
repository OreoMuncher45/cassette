package playlist

import (
	"os"
	"testing"

	"cassette/core/ytmusic"
)

func TestPlaylistOperations(t *testing.T) {
	// Use a temporary test file
	tmpFile, err := os.CreateTemp("", "playlists_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	store := &playlistStore{
		Playlists: []Playlist{},
	}
	if err := saveStore(store); err != nil {
		t.Fatal(err)
	}

	pl, err := CreatePlaylist("Chill Vibes")
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	if pl.Name != "Chill Vibes" {
		t.Errorf("pl.Name = %q, want %q", pl.Name, "Chill Vibes")
	}

	track := ytmusic.Track{
		VideoID: "vid123",
		Title:   "Cool Song",
		Artist:  "Great Artist",
	}

	if err := AddTrackToPlaylist(pl.ID, track); err != nil {
		t.Fatalf("AddTrackToPlaylist failed: %v", err)
	}

	// Verify track added
	fetched, err := GetPlaylist(pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}
	if len(fetched.Tracks) != 1 || fetched.Tracks[0].VideoID != "vid123" {
		t.Fatalf("expected 1 track with vid123, got: %+v", fetched.Tracks)
	}

	// Remove track
	if err := RemoveTrackFromPlaylist(pl.ID, "vid123"); err != nil {
		t.Fatalf("RemoveTrackFromPlaylist failed: %v", err)
	}
	fetched, _ = GetPlaylist(pl.ID)
	if len(fetched.Tracks) != 0 {
		t.Fatalf("expected 0 tracks after removal, got %d", len(fetched.Tracks))
	}

	// Delete playlist
	if err := DeletePlaylist(pl.ID); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}
	_, err = GetPlaylist(pl.ID)
	if err == nil {
		t.Fatal("expected error getting deleted playlist, got nil")
	}
}
