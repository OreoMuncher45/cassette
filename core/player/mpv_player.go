package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"cassette/core/logger"
	"cassette/core/utils"
)

// MpvPlayer drives audio playback through mpv's JSON IPC protocol over a UNIX socket.
type MpvPlayer struct {
	cmd        *exec.Cmd
	conn       net.Conn
	sockPath   string
	mu         sync.RWMutex
	playing    bool
	paused     bool
	positionMs int
	durationMs int
	volume     int
	requestID  int
	stopCh     chan struct{}
	endCh      chan struct{}
}

// NewMpvPlayer starts mpv in idle mode with JSON IPC and returns the player handle.
func NewMpvPlayer() (*MpvPlayer, error) {
	mpvBin, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}

	sockPath := filepath.Join(utils.SafeGetConfigDir(), "mpv.sock")

	// Remove stale socket
	_ = os.Remove(sockPath)

	cmd := exec.Command(mpvBin,
		"--idle",
		"--no-video",
		"--no-terminal",
		"--really-quiet",
		fmt.Sprintf("--input-ipc-server=%s", sockPath),
		"--volume=50",
		"--audio-display=no",
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mpv: %w", err)
	}

	// Wait for socket to appear
	var conn net.Conn
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
	}
	if conn == nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("mpv IPC socket not available after 5s")
	}

	p := &MpvPlayer{
		cmd:      cmd,
		conn:     conn,
		sockPath: sockPath,
		volume:   50,
		stopCh:   make(chan struct{}),
		endCh:    make(chan struct{}, 8),
	}

	// Register property observers with mpv IPC
	_ = p.sendCommand("observe_property", 1, "time-pos")
	_ = p.sendCommand("observe_property", 2, "duration")
	_ = p.sendCommand("observe_property", 3, "pause")

	go p.readLoop()
	go p.pollPosition()

	logger.Log.Info().Str("socket", sockPath).Msg("mpv player started")
	return p, nil
}

// EndChannel returns a channel that signals when the currently playing track finishes (EOF).
func (p *MpvPlayer) EndChannel() <-chan struct{} {
	return p.endCh
}

// Play loads and plays a URL (YouTube Music URL resolved by yt-dlp via mpv).
func (p *MpvPlayer) Play(url string) error {
	p.mu.Lock()
	p.positionMs = 0
	p.durationMs = 0
	p.playing = true
	p.paused = false
	p.mu.Unlock()

	return p.sendCommand("loadfile", url, "replace")
}

// Pause pauses playback.
func (p *MpvPlayer) Pause() error {
	err := p.setProperty("pause", true)
	if err == nil {
		p.mu.Lock()
		p.paused = true
		p.playing = false
		p.mu.Unlock()
	}
	return err
}

// Resume resumes playback.
func (p *MpvPlayer) Resume() error {
	err := p.setProperty("pause", false)
	if err == nil {
		p.mu.Lock()
		p.paused = false
		p.playing = true
		p.mu.Unlock()
	}
	return err
}

// Stop stops playback.
func (p *MpvPlayer) Stop() error {
	err := p.sendCommand("stop")
	if err == nil {
		p.mu.Lock()
		p.playing = false
		p.paused = false
		p.positionMs = 0
		p.mu.Unlock()
	}
	return err
}

// Seek seeks to an absolute position in seconds.
func (p *MpvPlayer) Seek(positionMs int) error {
	secs := float64(positionMs) / 1000.0
	return p.sendCommand("seek", secs, "absolute")
}

// SeekRelative seeks relative to current position.
func (p *MpvPlayer) SeekRelative(offsetMs int) error {
	secs := float64(offsetMs) / 1000.0
	return p.sendCommand("seek", secs, "relative")
}

// SetVolume sets volume (0-100).
func (p *MpvPlayer) SetVolume(percent int) error {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	err := p.setProperty("volume", percent)
	if err == nil {
		p.mu.Lock()
		p.volume = percent
		p.mu.Unlock()
	}
	return err
}

// GetVolume returns current volume.
func (p *MpvPlayer) GetVolume() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.volume
}

// PositionMs returns current playback position in milliseconds.
func (p *MpvPlayer) PositionMs() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.positionMs
}

// DurationMs returns current track duration in milliseconds.
func (p *MpvPlayer) DurationMs() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.durationMs
}

// IsPlaying returns true if audio is actively playing.
func (p *MpvPlayer) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.playing && !p.paused
}

// IsPaused returns true if playback is paused.
func (p *MpvPlayer) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paused
}

// Shutdown cleanly kills mpv and removes the socket.
func (p *MpvPlayer) Shutdown() {
	close(p.stopCh)

	if p.conn != nil {
		_ = p.sendCommand("quit")
		_ = p.conn.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		done := make(chan struct{})
		go func() {
			_ = p.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = p.cmd.Process.Kill()
			<-done
		}
	}
	_ = os.Remove(p.sockPath)
	logger.Log.Info().Msg("mpv player shut down")
}

type mpvCommand struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id,omitempty"`
}

func (p *MpvPlayer) sendCommandWithID(rid int, args ...interface{}) error {
	cmd := mpvCommand{
		Command:   args,
		RequestID: rid,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if p.conn == nil {
		return fmt.Errorf("mpv IPC connection closed")
	}

	_ = p.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = p.conn.Write(data)
	return err
}

func (p *MpvPlayer) sendCommand(args ...interface{}) error {
	p.mu.Lock()
	p.requestID++
	rid := p.requestID
	p.mu.Unlock()

	return p.sendCommandWithID(rid, args...)
}

func (p *MpvPlayer) setProperty(name string, value interface{}) error {
	return p.sendCommand("set_property", name, value)
}

// readLoop reads mpv IPC events to track playback state changes.
func (p *MpvPlayer) readLoop() {
	scanner := bufio.NewScanner(p.conn)
	for scanner.Scan() {
		select {
		case <-p.stopCh:
			return
		default:
		}

		line := scanner.Bytes()
		var msg map[string]interface{}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		// Handle property change events from observe_property
		if event, ok := msg["event"].(string); ok {
			switch event {
			case "property-change":
				name, _ := msg["name"].(string)
				switch name {
				case "time-pos":
					if val, ok := msg["data"].(float64); ok && val >= 0 {
						p.mu.Lock()
						p.positionMs = int(val * 1000)
						p.mu.Unlock()
					}
				case "duration":
					if val, ok := msg["data"].(float64); ok && val > 0 {
						p.mu.Lock()
						p.durationMs = int(val * 1000)
						p.mu.Unlock()
					}
				case "pause":
					if paused, ok := msg["data"].(bool); ok {
						p.mu.Lock()
						p.paused = paused
						p.playing = !paused
						p.mu.Unlock()
					}
				}
			case "end-file":
				p.mu.Lock()
				p.playing = false
				p.paused = false
				p.mu.Unlock()

				reason, _ := msg["reason"].(string)
				if reason == "eof" {
					select {
					case p.endCh <- struct{}{}:
					default:
					}
				}
			case "file-loaded":
				p.mu.Lock()
				p.playing = true
				p.paused = false
				p.mu.Unlock()
			case "pause":
				p.mu.Lock()
				p.paused = true
				p.playing = false
				p.mu.Unlock()
			case "unpause":
				p.mu.Lock()
				p.paused = false
				p.playing = true
				p.mu.Unlock()
			}
		}

		// Also handle explicit get_property response data
		if rid, ok := msg["request_id"].(float64); ok {
			if val, ok := msg["data"].(float64); ok {
				p.mu.Lock()
				if int(rid) == 101 && val >= 0 {
					p.positionMs = int(val * 1000)
				} else if int(rid) == 102 && val > 0 {
					p.durationMs = int(val * 1000)
				}
				p.mu.Unlock()
			}
		}
	}
}

// pollPosition periodically queries mpv for the current position and duration.
func (p *MpvPlayer) pollPosition() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.mu.RLock()
			active := p.playing
			p.mu.RUnlock()
			if !active {
				continue
			}

			// Query position with dedicated IDs
			_ = p.sendCommandWithID(101, "get_property", "time-pos")
			_ = p.sendCommandWithID(102, "get_property", "duration")
		}
	}
}
