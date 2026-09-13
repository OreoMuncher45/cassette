package player

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"cassette/core/logger"
	"github.com/zmb3/spotify/v2"
)

type Player struct {
	client     *spotify.Client
	deviceID   spotify.ID
	deviceName string
	mu         sync.RWMutex
	shuffled   bool
	volume     int
}

func NewPlayer(client *spotify.Client, deviceID spotify.ID, deviceName string) *Player {
	return &Player{
		client:     client,
		deviceID:   deviceID,
		deviceName: deviceName,
		volume:     50,
	}
}

func (p *Player) SetDevice(deviceID spotify.ID, deviceName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deviceID = deviceID
	p.deviceName = deviceName
	logger.Log.Info().Str("device_id", string(deviceID)).Str("device_name", deviceName).Msg("target playback device updated")
}

func (p *Player) GetDevice() (spotify.ID, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deviceID, p.deviceName
}

func (p *Player) playOptions() *spotify.PlayOptions {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.deviceID == "" {
		return nil
	}
	devID := p.deviceID
	return &spotify.PlayOptions{
		DeviceID: &devID,
	}
}

func (p *Player) GetDevices(ctx context.Context) ([]spotify.PlayerDevice, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("spotify client is not initialized")
	}
	devices, err := p.client.PlayerDevices(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get player devices")
		return nil, err
	}
	return devices, nil
}

func (p *Player) GetQueue(ctx context.Context) (*spotify.Queue, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("spotify client is not initialized")
	}
	return p.client.GetQueue(ctx)
}

func (p *Player) TransferPlayback(ctx context.Context, deviceID spotify.ID, play bool) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	logger.Log.Info().Str("device_id", string(deviceID)).Bool("play", play).Msg("transferring playback")
	err := p.client.TransferPlayback(ctx, deviceID, play)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to transfer playback")
		return err
	}
	return nil
}

func (p *Player) PlayTrack(ctx context.Context, uri string, contextURI string) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}

	opts := p.playOptions()
	if opts == nil {
		opts = &spotify.PlayOptions{}
	}

	if strings.HasPrefix(contextURI, "spotify:playlist:") || strings.HasPrefix(contextURI, "spotify:album:") || strings.HasPrefix(contextURI, "spotify:artist:") {
		ctxURI := spotify.URI(contextURI)
		opts.PlaybackContext = &ctxURI
		if uri != "" && uri != contextURI {
			opts.PlaybackOffset = &spotify.PlaybackOffset{URI: spotify.URI(uri)}
		}
	} else if uri != "" {
		opts.URIs = []spotify.URI{spotify.URI(uri)}
	}

	logger.Log.Info().Str("uri", uri).Str("context_uri", contextURI).Any("device", opts.DeviceID).Msg("playing track via web api")
	err := p.client.PlayOpt(ctx, opts)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to play track")
		return err
	}

	// Trigger endless song radio when playing an individual track (e.g. from search)
	if contextURI == "" && strings.HasPrefix(uri, "spotify:track:") {
		go p.queueSongRadio(uri)
	}

	return nil
}

func (p *Player) queueSongRadio(trackURI string) {
	trackID := strings.TrimPrefix(trackURI, "spotify:track:")
	if trackID == "" || trackID == trackURI {
		return
	}
	time.Sleep(600 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	track, err := p.client.GetTrack(ctx, spotify.ID(trackID))
	if err != nil || len(track.Artists) == 0 {
		return
	}

	artistID := track.Artists[0].ID
	topTracks, err := p.client.GetArtistsTopTracks(ctx, artistID, "from_token")
	if err != nil {
		topTracks, err = p.client.GetArtistsTopTracks(ctx, artistID, "US")
		if err != nil {
			return
		}
	}

	queued := 0
	for _, t := range topTracks {
		if t.ID == spotify.ID(trackID) {
			continue
		}
		if err := p.client.QueueSong(ctx, t.ID); err == nil {
			queued++
			if queued >= 8 {
				break
			}
		}
	}
	logger.Log.Info().Int("queued", queued).Str("seed_track", track.Name).Msg("queued endless song radio")
}

func (p *Player) PlayPause(ctx context.Context, currentlyPlaying bool) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	opts := p.playOptions()

	if currentlyPlaying {
		logger.Log.Info().Msg("pausing playback via web api")
		if err := p.client.PauseOpt(ctx, opts); err != nil {
			logger.Log.Error().Err(err).Msg("failed to pause playback")
			return err
		}
		return nil
	}

	logger.Log.Info().Msg("resuming playback via web api")
	if err := p.client.PlayOpt(ctx, opts); err != nil {
		logger.Log.Error().Err(err).Msg("failed to resume playback")
		return err
	}
	return nil
}

func (p *Player) Next(ctx context.Context) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	logger.Log.Info().Msg("skipping to next track via web api")
	if err := p.client.NextOpt(ctx, p.playOptions()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to skip to next track")
		return err
	}
	return nil
}

func (p *Player) Previous(ctx context.Context) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	logger.Log.Info().Msg("skipping to previous track via web api")
	if err := p.client.PreviousOpt(ctx, p.playOptions()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to skip to previous track")
		return err
	}
	return nil
}

func (p *Player) Seek(ctx context.Context, positionMs int, relative bool, currentPositionMs int) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	targetPosition := positionMs
	if relative {
		targetPosition = max(0, currentPositionMs+positionMs)
	}

	logger.Log.Info().Int("target_ms", targetPosition).Msg("seeking track via web api")
	if err := p.client.SeekOpt(ctx, targetPosition, p.playOptions()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to seek track")
		return err
	}
	return nil
}

func (p *Player) Shuffle(ctx context.Context, shuffle bool) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	logger.Log.Info().Bool("shuffle", shuffle).Msg("toggling shuffle via web api")
	if err := p.client.ShuffleOpt(ctx, shuffle, p.playOptions()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to toggle shuffle")
		return err
	}
	p.mu.Lock()
	p.shuffled = shuffle
	p.mu.Unlock()
	return nil
}

func (p *Player) Shuffled() bool {
	if p == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.shuffled
}

func (p *Player) SetVolume(ctx context.Context, volumePercent int) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("spotify client is not initialized")
	}
	targetPercent := max(0, min(100, volumePercent))
	logger.Log.Info().Int("volume_percent", targetPercent).Msg("setting volume via web api")
	if err := p.client.VolumeOpt(ctx, targetPercent, p.playOptions()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to set volume")
		return err
	}
	p.mu.Lock()
	p.volume = targetPercent
	p.mu.Unlock()
	return nil
}

func (p *Player) GetVolume() int {
	if p == nil {
		return 50
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.volume
}

func (p *Player) GetPlayerState(ctx context.Context) (*spotify.PlayerState, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("spotify client is not initialized")
	}
	state, err := p.client.PlayerState(ctx)
	if err != nil {
		return nil, err
	}
	if state != nil {
		p.mu.Lock()
		p.shuffled = state.ShuffleState
		if state.Device.Volume > 0 {
			p.volume = int(state.Device.Volume)
		}
		if state.Device.ID != "" {
			p.deviceID = state.Device.ID
			p.deviceName = state.Device.Name
		}
		p.mu.Unlock()
	}
	return state, nil
}

func (p *Player) GetDeviceID() spotify.ID {
	if p == nil {
		return ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deviceID
}

func (p *Player) GetDeviceName() string {
	if p == nil {
		return ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deviceName
}

func (p *Player) AutoSelectLocalDevice(ctx context.Context) (bool, error) {
	if p == nil || p.client == nil {
		return false, fmt.Errorf("spotify client is not initialized")
	}
	devices, err := p.client.PlayerDevices(ctx)
	if err != nil {
		return false, err
	}
	for _, dev := range devices {
		name := strings.ToLower(dev.Name)
		if strings.Contains(name, "cassette") {
			p.mu.Lock()
			p.deviceID = dev.ID
			p.deviceName = dev.Name
			p.mu.Unlock()
			if !dev.Active {
				_ = p.client.TransferPlayback(ctx, dev.ID, false)
			}
			logger.Log.Info().Str("device", dev.Name).Msg("auto-selected local Cassette audio sink")
			return true, nil
		}
	}
	return false, nil
}

func (p *Player) GetCurrentlyPlaying(ctx context.Context) (*spotify.CurrentlyPlaying, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("spotify client is not initialized")
	}
	return p.client.PlayerCurrentlyPlaying(ctx)
}

func IsNoActiveDeviceError(err error) bool {
	if err == nil {
		return false
	}
	var spotErr spotify.Error
	if errors.As(err, &spotErr) {
		if spotErr.Status == http.StatusNotFound {
			return true
		}
		if strings.Contains(strings.ToLower(spotErr.Message), "no active device") {
			return true
		}
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "no active device") ||
		strings.Contains(errMsg, "no_active_device") ||
		strings.Contains(errMsg, "device not found") ||
		strings.Contains(errMsg, "restricted device") ||
		strings.Contains(errMsg, "status 404") ||
		strings.Contains(errMsg, "status: 404") ||
		strings.Contains(errMsg, "404 not found")
}
