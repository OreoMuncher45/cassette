package nowplaying

import (
	"strings"
	"testing"

	"cassette/ui/v1/common"
)

func TestNowPlayingView(t *testing.T) {
	m := NewModel()
	m.SetSize(120, 12)
	m.SetSong(common.SongInfo{
		Title:    "Jonna",
		Artist:   "Westkust",
		Album:    "Last Forever",
		Position: 77000,
		Duration: 192000,
	})
	m.SetStatus(true, "cassette (PC)", false)
	m.SetArtwork("▀▀▀▀\n▀▀▀▀")

	view := m.View()
	if !strings.Contains(view, "Jonna") {
		t.Fatal("expected view to contain track title Jonna")
	}
	if !strings.Contains(view, "Westkust") {
		t.Fatal("expected view to contain artist Westkust")
	}
	if !strings.Contains(view, "01:17") {
		t.Fatal("expected view to contain position 01:17")
	}
	if !strings.Contains(view, "03:12") {
		t.Fatal("expected view to contain duration 03:12")
	}
	if !strings.Contains(view, "●") {
		t.Fatal("expected view to contain progress knob ●")
	}
}
