package queue

import (
	"strings"
	"testing"

	"github.com/zmb3/spotify/v2"
)

func TestQueueViewEmpty(t *testing.T) {
	m := NewModel()
	m.SetSize(30, 15)
	view := m.View()

	if !strings.Contains(view, "QUEUE") {
		t.Fatal("expected view to contain QUEUE header")
	}
	if !strings.Contains(view, "Queue is empty") {
		t.Fatal("expected view to state queue is empty")
	}
}

func TestQueueViewPopulated(t *testing.T) {
	m := NewModel()
	m.SetSize(30, 15)
	m.SetQueue(&spotify.Queue{
		Items: []spotify.FullTrack{
			{
				SimpleTrack: spotify.SimpleTrack{
					Name:     "Upcoming Track 1",
					Duration: 215000,
					Artists: []spotify.SimpleArtist{
						{Name: "Artist One"},
					},
				},
			},
			{
				SimpleTrack: spotify.SimpleTrack{
					Name:     "Upcoming Track 2",
					Duration: 180000,
					Artists: []spotify.SimpleArtist{
						{Name: "Artist Two"},
					},
				},
			},
		},
	})

	view := m.View()
	if !strings.Contains(view, "Upcoming Track 1") {
		t.Fatal("expected queue view to contain Upcoming Track 1")
	}
	if !strings.Contains(view, "Artist One") {
		t.Fatal("expected queue view to contain Artist One")
	}
}
