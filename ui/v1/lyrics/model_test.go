package lyrics

import (
	"strings"
	"testing"

	"cassette/core/lyrics"
)

func TestLyricsViewEmpty(t *testing.T) {
	m := NewModel()
	m.SetSize(30, 15)
	view := m.View()

	if !strings.Contains(view, "LYRICS") {
		t.Fatal("expected view to contain LYRICS header")
	}
	if !strings.Contains(view, "No lyrics available") {
		t.Fatal("expected view to indicate no lyrics")
	}
}

func TestLyricsViewSynced(t *testing.T) {
	m := NewModel()
	m.SetSize(30, 15)
	m.SetLyrics(&lyrics.Lyrics{
		TrackName:  "Test Song",
		ArtistName: "Test Artist",
		HasSynced:  true,
		SyncedLyrics: []lyrics.LyricLine{
			{TimeMs: 1000, Text: "Intro line"},
			{TimeMs: 5000, Text: "Singing line"},
			{TimeMs: 10000, Text: "Ending line"},
		},
	})
	m.SetPosition(5500)

	view := m.View()
	if !strings.Contains(view, "Singing line") {
		t.Fatal("expected active line to appear in view")
	}
	if !strings.Contains(view, "▶") {
		t.Fatal("expected active indicator ▶ in synced lyrics view")
	}
}
