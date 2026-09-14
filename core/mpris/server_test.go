package mpris

import (
	"os"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestBuildMetadata(t *testing.T) {
	state := PlaybackState{
		Playing:    true,
		Title:      "Around the World",
		Artist:     "Daft Punk, Romanthony",
		Album:      "Homework",
		ArtURL:     "https://i.scdn.co/image/ab67616d0000b273test",
		TrackID:    "4cOdK2wGLETKBW3PvgPWqT",
		PositionMs: 30000,
		DurationMs: 240000,
		Volume:     75,
		Shuffled:   true,
	}

	meta := BuildMetadata(state)

	if val, ok := meta["xesam:title"]; !ok || val.Value().(string) != "Around the World" {
		t.Errorf("expected xesam:title to be 'Around the World', got %v", val)
	}

	if val, ok := meta["xesam:album"]; !ok || val.Value().(string) != "Homework" {
		t.Errorf("expected xesam:album to be 'Homework', got %v", val)
	}

	if val, ok := meta["xesam:artist"]; !ok {
		t.Errorf("missing xesam:artist")
	} else {
		artists, ok := val.Value().([]string)
		if !ok || len(artists) != 2 || artists[0] != "Daft Punk" || artists[1] != "Romanthony" {
			t.Errorf("unexpected xesam:artist: %v", val.Value())
		}
	}

	if val, ok := meta["mpris:length"]; !ok || val.Value().(int64) != 240000000 {
		t.Errorf("expected mpris:length to be 240000000 us, got %v", val)
	}

	if val, ok := meta["mpris:trackid"]; !ok {
		t.Errorf("missing mpris:trackid")
	} else {
		objPath, ok := val.Value().(dbus.ObjectPath)
		if !ok || objPath != dbus.ObjectPath("/org/mpris/MediaPlayer2/Track/4cOdK2wGLETKBW3PvgPWqT") {
			t.Errorf("unexpected mpris:trackid: %v", val.Value())
		}
	}
}

func TestDbusServerLifecycle(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("skipping dbus integration test: DBUS_SESSION_BUS_ADDRESS not set")
	}

	playPauseCalled := false
	cb := Callbacks{
		OnPlayPause: func() error {
			playPauseCalled = true
			return nil
		},
	}

	srv, err := NewServer(cb)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	// Update playback state
	srv.UpdatePlayback(PlaybackState{
		Playing:    true,
		Title:      "Test Song",
		Artist:     "Test Artist",
		PositionMs: 5000,
		DurationMs: 180000,
		Volume:     80,
	})

	// Test handler directly
	h := &playerHandler{srv: srv}
	if err := h.PlayPause(); err != nil {
		t.Errorf("h.PlayPause failed: %v", err)
	}
	if !playPauseCalled {
		t.Errorf("expected OnPlayPause to be called")
	}

	if err := srv.Close(); err != nil {
		t.Errorf("srv.Close failed: %v", err)
	}
}
