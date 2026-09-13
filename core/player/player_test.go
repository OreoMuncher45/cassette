package player

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

func newMockSpotifyPlayer(t *testing.T, handler http.Handler) (*Player, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	httpClient := &http.Client{
		Transport: &mockTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		},
	}
	spotClient := spotify.New(httpClient, spotify.WithBaseURL(server.URL+"/v1/"))
	p := NewPlayer(spotClient, "dev-123", "Test Device")
	return p, server.Close
}

type mockTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	targetURL, _ := req.URL.Parse(m.baseURL + req.URL.Path)
	if req.URL.RawQuery != "" {
		targetURL.RawQuery = req.URL.RawQuery
	}
	clone.URL = targetURL
	return m.base.RoundTrip(clone)
}

func TestPlayerPlayTrackWithContext(t *testing.T) {
	var called bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/me/player/play" && r.Method == http.MethodPut {
			called = true
			if r.URL.Query().Get("device_id") != "dev-123" {
				t.Fatalf("device_id = %q, want dev-123", r.URL.Query().Get("device_id"))
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})

	p, cleanup := newMockSpotifyPlayer(t, handler)
	defer cleanup()

	err := p.PlayTrack(context.Background(), "spotify:track:t1", "spotify:playlist:p1")
	if err != nil {
		t.Fatalf("PlayTrack error = %v, want nil", err)
	}
	if !called {
		t.Fatal("expected PUT /v1/me/player/play to be called")
	}
}

func TestPlayerPlayPause(t *testing.T) {
	var pauseCalled, playCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me/player/pause":
			pauseCalled = true
			w.WriteHeader(http.StatusNoContent)
		case "/v1/me/player/play":
			playCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	p, cleanup := newMockSpotifyPlayer(t, handler)
	defer cleanup()

	if err := p.PlayPause(context.Background(), true); err != nil {
		t.Fatalf("PlayPause(true) error = %v", err)
	}
	if !pauseCalled {
		t.Fatal("expected pause to be called")
	}

	if err := p.PlayPause(context.Background(), false); err != nil {
		t.Fatalf("PlayPause(false) error = %v", err)
	}
	if !playCalled {
		t.Fatal("expected play to be called")
	}
}

func TestPlayerSeekAndVolume(t *testing.T) {
	var seekCalled, volumeCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me/player/seek":
			seekCalled = true
			if r.URL.Query().Get("position_ms") != "5000" {
				t.Fatalf("position_ms = %q, want 5000", r.URL.Query().Get("position_ms"))
			}
			w.WriteHeader(http.StatusNoContent)
		case "/v1/me/player/volume":
			volumeCalled = true
			if r.URL.Query().Get("volume_percent") != "75" {
				t.Fatalf("volume_percent = %q, want 75", r.URL.Query().Get("volume_percent"))
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	p, cleanup := newMockSpotifyPlayer(t, handler)
	defer cleanup()

	if err := p.Seek(context.Background(), 5000, false, 0); err != nil {
		t.Fatalf("Seek error = %v", err)
	}
	if !seekCalled {
		t.Fatal("expected seek to be called")
	}

	if err := p.SetVolume(context.Background(), 75); err != nil {
		t.Fatalf("SetVolume error = %v", err)
	}
	if !volumeCalled {
		t.Fatal("expected volume to be called")
	}
	if p.GetVolume() != 75 {
		t.Fatalf("GetVolume = %d, want 75", p.GetVolume())
	}
}

func TestPlayerNextPreviousShuffle(t *testing.T) {
	var nextCalled, prevCalled, shuffleCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me/player/next":
			nextCalled = true
			w.WriteHeader(http.StatusNoContent)
		case "/v1/me/player/previous":
			prevCalled = true
			w.WriteHeader(http.StatusNoContent)
		case "/v1/me/player/shuffle":
			shuffleCalled = true
			if r.URL.Query().Get("state") != "true" {
				t.Fatalf("shuffle state = %q, want true", r.URL.Query().Get("state"))
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	p, cleanup := newMockSpotifyPlayer(t, handler)
	defer cleanup()

	if err := p.Next(context.Background()); err != nil {
		t.Fatalf("Next error = %v", err)
	}
	if !nextCalled {
		t.Fatal("expected next to be called")
	}

	if err := p.Previous(context.Background()); err != nil {
		t.Fatalf("Previous error = %v", err)
	}
	if !prevCalled {
		t.Fatal("expected previous to be called")
	}

	if err := p.Shuffle(context.Background(), true); err != nil {
		t.Fatalf("Shuffle error = %v", err)
	}
	if !shuffleCalled {
		t.Fatal("expected shuffle to be called")
	}
	if !p.Shuffled() {
		t.Fatal("Shuffled() = false, want true")
	}
}

func TestPlayerDevicesAndTransfer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me/player/devices":
			resp := struct {
				Devices []spotify.PlayerDevice `json:"devices"`
			}{
				Devices: []spotify.PlayerDevice{
					{ID: "d1", Name: "Desktop", Type: "Computer", Active: true},
					{ID: "d2", Name: "Phone", Type: "Smartphone", Active: false},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case "/v1/me/player":
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	p, cleanup := newMockSpotifyPlayer(t, handler)
	defer cleanup()

	devs, err := p.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices error = %v", err)
	}
	if len(devs) != 2 {
		t.Fatalf("len(devs) = %d, want 2", len(devs))
	}

	if err := p.TransferPlayback(context.Background(), "d2", false); err != nil {
		t.Fatalf("TransferPlayback error = %v", err)
	}
}

func TestIsNoActiveDeviceError(t *testing.T) {
	if IsNoActiveDeviceError(nil) {
		t.Fatal("nil error should not be no active device error")
	}

	spot404 := spotify.Error{Status: http.StatusNotFound, Message: "No active device found"}
	if !IsNoActiveDeviceError(spot404) {
		t.Fatal("spotify 404 should be recognized as no active device error")
	}

	genericErr := errors.New("Player command failed: No active device found (404)")
	if !IsNoActiveDeviceError(genericErr) {
		t.Fatal("generic error containing no active device should be recognized")
	}

	otherErr := errors.New("unauthorized: bad token")
	if IsNoActiveDeviceError(otherErr) {
		t.Fatal("unauthorized error should not be recognized as no active device")
	}
}

// Dummy oauth2 token import check
var _ = oauth2.Token{}
