package mpris

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"sync"

	"cassette/core/logger"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	ObjectPath           = "/org/mpris/MediaPlayer2"
	InterfaceRoot        = "org.mpris.MediaPlayer2"
	InterfacePlayer      = "org.mpris.MediaPlayer2.Player"
	BusNameBase          = "org.mpris.MediaPlayer2.cassette"
	DefaultTrackIDPrefix = "/org/mpris/MediaPlayer2/Track/"
	NoTrackPath          = "/org/mpris/MediaPlayer2/TrackList/NoTrack"
)

var cleanPathRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Callbacks holds function hooks triggered by incoming MPRIS D-Bus method invocations.
type Callbacks struct {
	OnPlayPause   func() error
	OnPlay        func() error
	OnPause       func() error
	OnNext        func() error
	OnPrevious    func() error
	OnStop        func() error
	OnSeek        func(offsetUs int64) error
	OnSetPosition func(trackID string, positionUs int64) error
	OnOpenUri     func(uri string) error
	OnSetVolume   func(volumePercent int) error
	OnSetShuffle  func(shuffle bool) error
	OnQuit        func() error
	OnRaise       func() error
}

// PlaybackState represents snapshot data broadcasted to MPRIS clients (earbuds, media keys, desktop).
type PlaybackState struct {
	Playing    bool
	Title      string
	Artist     string
	Album      string
	ArtURL     string
	TrackID    string
	PositionMs int
	DurationMs int
	Volume     int // 0 to 100
	Shuffled   bool
}

// Server manages the D-Bus connection and MPRIS interfaces.
type Server struct {
	conn    *dbus.Conn
	props   *prop.Properties
	busName string
	cb      Callbacks
	mu      sync.RWMutex
	closed  bool
	state   PlaybackState
}

// NewServer connects to the D-Bus session bus and exports the MPRIS 2.2 player interfaces.
func NewServer(cb Callbacks) (*Server, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	srv := &Server{
		conn: conn,
		cb:   cb,
	}

	rootH := &rootHandler{srv: srv}
	playerH := &playerHandler{srv: srv}

	// Build properties map
	propsMap := prop.Map{
		InterfaceRoot: {
			"CanQuit":             {Value: true, Writable: false, Emit: prop.EmitConst},
			"Fullscreen":          {Value: false, Writable: false, Emit: prop.EmitFalse},
			"CanSetFullscreen":     {Value: false, Writable: false, Emit: prop.EmitConst},
			"CanRaise":            {Value: false, Writable: false, Emit: prop.EmitConst},
			"HasTrackList":        {Value: false, Writable: false, Emit: prop.EmitConst},
			"Identity":            {Value: "Cassette", Writable: false, Emit: prop.EmitConst},
			"DesktopEntry":        {Value: "cassette", Writable: false, Emit: prop.EmitConst},
			"SupportedUriSchemes": {Value: []string{"spotify"}, Writable: false, Emit: prop.EmitConst},
			"SupportedMimeTypes":  {Value: []string{}, Writable: false, Emit: prop.EmitConst},
		},
		InterfacePlayer: {
			"PlaybackStatus": {Value: "Stopped", Writable: false, Emit: prop.EmitTrue},
			"LoopStatus":     {Value: "None", Writable: true, Emit: prop.EmitTrue},
			"Rate":           {Value: 1.0, Writable: false, Emit: prop.EmitFalse},
			"Shuffle": {
				Value:    false,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					shuf, ok := c.Value.(bool)
					if !ok {
						return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"Shuffle must be a boolean"})
					}
					srv.mu.Lock()
					srv.state.Shuffled = shuf
					srv.mu.Unlock()
					if srv.cb.OnSetShuffle != nil {
						_ = srv.cb.OnSetShuffle(shuf)
					}
					return nil
				},
			},
			"Metadata": {Value: map[string]dbus.Variant{}, Writable: false, Emit: prop.EmitTrue},
			"Volume": {
				Value:    0.5,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					vol, ok := c.Value.(float64)
					if !ok {
						return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"Volume must be a float"})
					}
					if vol < 0.0 {
						vol = 0.0
					} else if vol > 1.0 {
						vol = 1.0
					}
					c.Value = vol
					pct := int(math.Round(vol * 100))
					srv.mu.Lock()
					srv.state.Volume = pct
					srv.mu.Unlock()
					if srv.cb.OnSetVolume != nil {
						_ = srv.cb.OnSetVolume(pct)
					}
					return nil
				},
			},
			"Position":       {Value: int64(0), Writable: false, Emit: prop.EmitFalse},
			"MinimumRate":    {Value: 1.0, Writable: false, Emit: prop.EmitConst},
			"MaximumRate":    {Value: 1.0, Writable: false, Emit: prop.EmitConst},
			"CanControl":     {Value: true, Writable: false, Emit: prop.EmitConst},
			"CanPlay":        {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanPause":       {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanSeek":        {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanGoNext":      {Value: true, Writable: false, Emit: prop.EmitFalse},
			"CanGoPrevious":  {Value: true, Writable: false, Emit: prop.EmitFalse},
		},
	}

	props, err := prop.Export(conn, ObjectPath, propsMap)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export properties: %w", err)
	}
	srv.props = props

	// Export methods
	if err := conn.Export(rootH, ObjectPath, InterfaceRoot); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export root interface: %w", err)
	}
	if err := conn.Export(playerH, ObjectPath, InterfacePlayer); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export player interface: %w", err)
	}

	// Export Introspectable
	node := &introspect.Node{
		Name: ObjectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       InterfaceRoot,
				Methods:    introspect.Methods(rootH),
				Properties: props.Introspection(InterfaceRoot),
			},
			{
				Name:       InterfacePlayer,
				Methods:    introspect.Methods(playerH),
				Properties: props.Introspection(InterfacePlayer),
			},
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(node), ObjectPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		logger.Log.Warn().Err(err).Msg("failed to export introspection")
	}

	// Request well-known bus name
	busName := BusNameBase
	reply, err := conn.RequestName(busName, dbus.NameFlagReplaceExisting)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		busName = fmt.Sprintf("%s.instance%d", BusNameBase, os.Getpid())
		_, _ = conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	}
	srv.busName = busName
	logger.Log.Info().Str("bus_name", busName).Msg("MPRIS D-Bus service registered")

	return srv, nil
}

// UpdatePlayback pushes new playback status, metadata, volume, and progress across D-Bus.
func (s *Server) UpdatePlayback(state PlaybackState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.props == nil {
		return
	}
	s.state = state

	status := "Stopped"
	if state.Playing {
		status = "Playing"
	} else if state.Title != "" {
		status = "Paused"
	}
	s.props.SetMust(InterfacePlayer, "PlaybackStatus", status)

	meta := BuildMetadata(state)
	s.props.SetMust(InterfacePlayer, "Metadata", meta)
	s.props.SetMust(InterfacePlayer, "Position", int64(state.PositionMs)*1000)
	s.props.SetMust(InterfacePlayer, "Volume", float64(state.Volume)/100.0)
	s.props.SetMust(InterfacePlayer, "Shuffle", state.Shuffled)
}

// BuildMetadata converts PlaybackState into the MPRIS-compliant map[string]dbus.Variant.
func BuildMetadata(state PlaybackState) map[string]dbus.Variant {
	meta := make(map[string]dbus.Variant)

	trackIDPath := NoTrackPath
	if state.TrackID != "" {
		cleanID := cleanPathRegex.ReplaceAllString(state.TrackID, "_")
		trackIDPath = DefaultTrackIDPrefix + cleanID
	} else if state.Title != "" {
		trackIDPath = DefaultTrackIDPrefix + "current"
	}
	meta["mpris:trackid"] = dbus.MakeVariant(dbus.ObjectPath(trackIDPath))

	if state.Title != "" {
		meta["xesam:title"] = dbus.MakeVariant(state.Title)
	}

	if state.Artist != "" {
		parts := strings.Split(state.Artist, ", ")
		meta["xesam:artist"] = dbus.MakeVariant(parts)
	}

	if state.Album != "" {
		meta["xesam:album"] = dbus.MakeVariant(state.Album)
	}

	if state.ArtURL != "" {
		meta["mpris:artUrl"] = dbus.MakeVariant(state.ArtURL)
	}

	if state.DurationMs > 0 {
		meta["mpris:length"] = dbus.MakeVariant(int64(state.DurationMs) * 1000)
	}

	if state.TrackID != "" {
		meta["xesam:url"] = dbus.MakeVariant("https://open.spotify.com/track/" + state.TrackID)
	}

	return meta
}

// Close releases the bus name and disconnects from D-Bus.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if s.conn != nil {
		if s.busName != "" {
			_, _ = s.conn.ReleaseName(s.busName)
		}
		_ = s.conn.Close()
	}
	return nil
}

// rootHandler implements org.mpris.MediaPlayer2
type rootHandler struct {
	srv *Server
}

func (h *rootHandler) Raise() *dbus.Error {
	if h.srv.cb.OnRaise != nil {
		_ = h.srv.cb.OnRaise()
	}
	return nil
}

func (h *rootHandler) Quit() *dbus.Error {
	if h.srv.cb.OnQuit != nil {
		_ = h.srv.cb.OnQuit()
	}
	return nil
}

// playerHandler implements org.mpris.MediaPlayer2.Player
type playerHandler struct {
	srv *Server
}

func (h *playerHandler) Next() *dbus.Error {
	if h.srv.cb.OnNext != nil {
		if err := h.srv.cb.OnNext(); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) Previous() *dbus.Error {
	if h.srv.cb.OnPrevious != nil {
		if err := h.srv.cb.OnPrevious(); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) Pause() *dbus.Error {
	if h.srv.cb.OnPause != nil {
		if err := h.srv.cb.OnPause(); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) PlayPause() *dbus.Error {
	if h.srv.cb.OnPlayPause != nil {
		if err := h.srv.cb.OnPlayPause(); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) Stop() *dbus.Error {
	if h.srv.cb.OnStop != nil {
		if err := h.srv.cb.OnStop(); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) Play() *dbus.Error {
	if h.srv.cb.OnPlay != nil {
		if err := h.srv.cb.OnPlay(); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) Seek(offsetUs int64) *dbus.Error {
	if h.srv.cb.OnSeek != nil {
		if err := h.srv.cb.OnSeek(offsetUs); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) SetPosition(trackID dbus.ObjectPath, positionUs int64) *dbus.Error {
	if h.srv.cb.OnSetPosition != nil {
		if err := h.srv.cb.OnSetPosition(string(trackID), positionUs); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (h *playerHandler) OpenUri(uri string) *dbus.Error {
	if h.srv.cb.OnOpenUri != nil {
		if err := h.srv.cb.OnOpenUri(uri); err != nil {
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}
