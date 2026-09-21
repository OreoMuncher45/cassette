package playlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cassette/core/logger"
	"cassette/core/utils"
	"cassette/core/ytmusic"
)

// Playlist represents a user-created or default playlist.
type Playlist struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Tracks    []ytmusic.Track `json:"tracks"`
	CreatedAt time.Time       `json:"created_at"`
}

type playlistStore struct {
	Playlists []Playlist `json:"playlists"`
}

var (
	mu sync.RWMutex
)

func getStorePath() string {
	return filepath.Join(utils.SafeGetConfigDir(), "playlists.json")
}

// loadStore reads the playlists from disk.
func loadStore() (*playlistStore, error) {
	path := getStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize default favorites playlist
			store := &playlistStore{
				Playlists: []Playlist{
					{
						ID:        "favorites",
						Name:      "★ Favorites",
						Tracks:    []ytmusic.Track{},
						CreatedAt: time.Now(),
					},
				},
			}
			_ = saveStore(store)
			return store, nil
		}
		return nil, err
	}

	var store playlistStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

// saveStore writes the playlists to disk.
func saveStore(store *playlistStore) error {
	path := getStorePath()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetPlaylists returns all saved playlists.
func GetPlaylists() []Playlist {
	mu.RLock()
	defer mu.RUnlock()

	store, err := loadStore()
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to load playlists")
		return nil
	}
	return store.Playlists
}

// GetPlaylist retrieves a playlist by ID.
func GetPlaylist(id string) (*Playlist, error) {
	mu.RLock()
	defer mu.RUnlock()

	store, err := loadStore()
	if err != nil {
		return nil, err
	}

	for _, p := range store.Playlists {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("playlist %q not found", id)
}

// CreatePlaylist creates a new local playlist.
func CreatePlaylist(name string) (*Playlist, error) {
	mu.Lock()
	defer mu.Unlock()

	store, err := loadStore()
	if err != nil {
		store = &playlistStore{}
	}

	id := fmt.Sprintf("pl_%d", time.Now().UnixNano())
	pl := Playlist{
		ID:        id,
		Name:      name,
		Tracks:    []ytmusic.Track{},
		CreatedAt: time.Now(),
	}

	store.Playlists = append(store.Playlists, pl)
	if err := saveStore(store); err != nil {
		return nil, err
	}

	logger.Log.Info().Str("name", name).Str("id", id).Msg("playlist created")
	return &pl, nil
}

// AddTrackToPlaylist adds a track to the specified playlist.
func AddTrackToPlaylist(playlistID string, track ytmusic.Track) error {
	mu.Lock()
	defer mu.Unlock()

	store, err := loadStore()
	if err != nil {
		return err
	}

	for i := range store.Playlists {
		if store.Playlists[i].ID == playlistID {
			// Prevent duplicates
			for _, t := range store.Playlists[i].Tracks {
				if t.VideoID == track.VideoID {
					return nil // already in playlist
				}
			}
			store.Playlists[i].Tracks = append(store.Playlists[i].Tracks, track)
			return saveStore(store)
		}
	}
	return fmt.Errorf("playlist %q not found", playlistID)
}

// RemoveTrackFromPlaylist removes a track from a playlist by video ID.
func RemoveTrackFromPlaylist(playlistID, videoID string) error {
	mu.Lock()
	defer mu.Unlock()

	store, err := loadStore()
	if err != nil {
		return err
	}

	for i := range store.Playlists {
		if store.Playlists[i].ID == playlistID {
			var newTracks []ytmusic.Track
			for _, t := range store.Playlists[i].Tracks {
				if t.VideoID != videoID {
					newTracks = append(newTracks, t)
				}
			}
			store.Playlists[i].Tracks = newTracks
			return saveStore(store)
		}
	}
	return fmt.Errorf("playlist %q not found", playlistID)
}

// DeletePlaylist deletes a playlist by ID.
func DeletePlaylist(playlistID string) error {
	mu.Lock()
	defer mu.Unlock()

	store, err := loadStore()
	if err != nil {
		return err
	}

	var newPlaylists []Playlist
	found := false
	for _, p := range store.Playlists {
		if p.ID == playlistID {
			found = true
			continue
		}
		newPlaylists = append(newPlaylists, p)
	}

	if !found {
		return fmt.Errorf("playlist %q not found", playlistID)
	}

	store.Playlists = newPlaylists
	return saveStore(store)
}
