package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"cassette/core/artwork"
	"cassette/core/auth"
	"cassette/core/logger"
	corelyrics "cassette/core/lyrics"
	"cassette/core/mpris"
	coreplayer "cassette/core/player"
	"cassette/core/theme"
	"cassette/core/ticker"
	"cassette/core/utils"
	"cassette/core/ytmusic"
	"cassette/spotify"
	uiauth "cassette/ui/v1/auth"
	"cassette/ui/v1/common"
	"cassette/ui/v1/devices"
	"cassette/ui/v1/keybinds"
	"cassette/ui/v1/mediacenter"
	"cassette/ui/v1/player"
	"cassette/ui/v1/settings"
	spotapi "github.com/zmb3/spotify/v2"
)

type Model struct {
	authModel          *uiauth.Model
	devicesModel       *devices.Model
	devicePickerOpen   bool
	keybindsModel      keybinds.Model
	settingsModel      settings.Model
	playing            bool
	playerReady        bool
	isFocused          bool
	songInfo           common.SongInfo
	volumeInfo         common.VolumeInfo
	volumeOverlayUntil time.Time
	fatalErr           error
	player             *coreplayer.Player
	mpvPlayer          *coreplayer.MpvPlayer
	localDaemon        *coreplayer.LocalDaemon
	spotifyClient      *spotify.SpotifyClient
	mediaCenter        mediacenter.Model
	lastLyricsTrack    string
	lastLyricsArtist   string
	lastArtworkURL     string
	lastArtCols        int
	lastArtRows        int
	lastTrackID        string
	mprisServer        *mpris.Server
	program            *tea.Program
	width              int
	height             int
	help               help.Model
	keys               *common.AppKeyMap
	requestHandlers    map[common.MediaRequestKind]func(common.MediaRequest) tea.Cmd
	welcomeModel       *WelcomeModel
	ytQueue            []ytmusic.Track
	ytQueueIndex       int
}

type nextTrackOkMsg struct{}
type prevTrackOkMsg struct{}

type mediaLoadedMsg struct {
	entities   []common.Entity
	kind       common.ListKind
	pagination common.PaginationInfo
	request    common.MediaRequest
}

type mediaLoadErrMsg struct {
	err     error
	request common.MediaRequest
}

type playTrackErrMsg struct {
	err       error
	panelKind common.ListKind
}

type playTrackOkMsg struct {
	panelKind common.ListKind
}

type startupCompleteMsg struct{}

type mprisPollStateMsg struct{}

type playerStateMsg struct {
	state *spotapi.PlayerState
	err   error
}

type ytPlayerStateMsg struct {
	positionMs int
	durationMs int
	playing    bool
}

type ytRadioLoadedMsg struct {
	seedVideoID string
	tracks      []ytmusic.Track
	err         error
}

type ytRadioExtendedMsg struct {
	tracks []ytmusic.Track
}

type ytTrackEndedMsg struct{}

type playPauseOkMsg struct {
	playing bool
}

type volumeChangedMsg struct {
	volumeInfo common.VolumeInfo
}

type transportErrMsg struct {
	err    error
	action string
}

type shuffleOkMsg struct {
	shuffled bool
}

type fatalErrMsg struct {
	err error
}

type fatalQuitMsg struct{}

type lyricsLoadedMsg struct {
	track  string
	artist string
	lyrics *corelyrics.Lyrics
	err    error
}

type queueLoadedMsg struct {
	queue *spotapi.Queue
	err   error
}

type artworkLoadedMsg struct {
	imageURL string
	ansi     string
	err      error
}

func NewModel() *Model {
	keys := common.NewAppKeyMap()
	model := &Model{
		authModel:     uiauth.NewModel(),
		devicesModel:  devices.NewModel(false),
		keybindsModel: keybinds.NewModel(keys),
		settingsModel: settings.NewModel(),
		mediaCenter:   mediacenter.NewModel(keys),
		help:          newHelpModel(),
		keys:          keys,
		volumeInfo:    common.VolumeInfo{Volume: 50, Max: 100},
		isFocused:     true,
	}
	model.requestHandlers = map[common.MediaRequestKind]func(common.MediaRequest) tea.Cmd{
		common.GetUserPlaylists:   model.handleGetUserPlaylists,
		common.GetSavedTracks:     model.handleGetSavedTracks,
		common.GetSavedAlbums:     model.handleGetSavedAlbums,
		common.GetFollowedArtists: model.handleGetFollowedArtists,
		common.SearchPlaylists:    model.handleSearchPlaylists,
		common.SearchTracks:       model.handleSearchTracks,
		common.SearchAlbums:       model.handleSearchAlbums,
		common.SearchArtists:      model.handleSearchArtists,
		common.GetPlaylistTracks:  model.handleGetPlaylistTracks,
		common.GetArtistAlbums:    model.handleGetArtistAlbums,
		common.GetAlbumTracks:     model.handleGetAlbumTracks,
		common.PlayTrack:          model.handlePlayTrackRequest,
	}
	return model
}

func newHelpModel() help.Model {
	h := help.New()
	h.ShowAll = false
	return h
}

func Run() error {
	model := NewModel()
	p := tea.NewProgram(model)
	model.program = p
	_, err := p.Run()
	model.shutdown()
	return err
}

func (m *Model) Init() tea.Cmd {
	cmd := func() tea.Msg {
		err := m.start()
		if err != nil && m.authModel.State() == uiauth.Authenticated {
			return fatalErrMsg{err: err}
		}
		if m.authModel.State() == uiauth.NeedsAuth {
			return tea.Msg(m.authModel.State())
		}
		return startupCompleteMsg{}
	}
	return tea.Batch(cmd, ticker.StartTicker())
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
	m.help.SetWidth(width)
	if m.authModel != nil {
		m.authModel.SetSize(width, height)
	}
	if m.devicesModel != nil {
		m.devicesModel.SetSize(width, height)
	}
	m.keybindsModel.SetSize(width, height)
	m.settingsModel.SetSize(width, height)
}

func (m *Model) shutdown() {
	if m.mprisServer != nil {
		_ = m.mprisServer.Close()
	}
	if m.localDaemon != nil {
		m.localDaemon.Stop()
	}
	if m.mpvPlayer != nil {
		m.mpvPlayer.Shutdown()
	}
}

func (m *Model) initYouTubeMusic() {
	if m.authModel != nil {
		m.authModel.SetState(uiauth.Authenticated)
	}
	if m.mpvPlayer == nil {
		mpv, err := coreplayer.NewMpvPlayer()
		if err != nil {
			logger.Log.Warn().Err(err).Msg("failed to start mpv player daemon")
		} else {
			m.mpvPlayer = mpv
			m.volumeInfo = common.VolumeInfo{Volume: mpv.GetVolume(), Max: 100}
		}
	}
	m.playerReady = true
	m.initMpris()
	m.initWelcomePopup()
}

func (m *Model) switchToYouTubeMusic() {
	cfg := utils.GetConfig()
	cfg.DefaultSource = "ytmusic"
	_ = utils.SaveConfig(cfg)
	m.initYouTubeMusic()
	m.mediaCenter.SetDisplay("YouTube Music (Press / to search)")
	m.updatePlayerStatus()
}

func (m *Model) start() error {
	ctx := context.Background()
	var err error
	m.authModel = uiauth.NewModel()
	if m.width != 0 || m.height != 0 {
		m.authModel.SetSize(m.width, m.height)
		m.devicesModel.SetSize(m.width, m.height)
	}

	// YouTube Music mode requires ZERO credentials — skip Spotify auth completely
	if utils.IsYouTubeMusicMode() {
		m.initYouTubeMusic()
		return nil
	}

	m.spotifyClient, err = spotify.NewSpotifyClient(ctx, m.authModel.Authenticator())
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.SetState(uiauth.NeedsAuth)
		}
		logger.Log.Error().Err(err).Msg("failed to create spotify client")
		return err
	}

	userID, err := m.spotifyClient.GetUserID(ctx)
	logger.Log.Info().Str("user id", userID).Msg("got user id")
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.SetState(uiauth.NeedsAuth)
		}
		return err
	}

	token, err := auth.New().GetAuthToken(ctx)
	if err != nil || token == nil {
		m.authModel.SetState(uiauth.NeedsAuth)
		return err
	}

	m.player = coreplayer.NewPlayer(m.spotifyClient.RawClient(), "", "")
	m.initMpris()

	// Fetch initial devices
	devs, err := m.player.GetDevices(ctx)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to get player devices on startup")
		if spotify.IsAuthError(err) {
			m.authModel.SetState(uiauth.NeedsAuth)
			return err
		}
	}

	m.localDaemon = coreplayer.StartLocalDaemon()
	m.devicesModel.SetDevices(devs)
	// If a device is already active or local device exists, don't force picker open
	hasActive := false
	for _, d := range devs {
		if d.Active {
			hasActive = true
			break
		}
	}
	if !hasActive && len(devs) > 0 {
		m.devicePickerOpen = true
	}

	return nil
}

func (m *Model) fetchDevicesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.player == nil {
			return devices.DevicesLoadedMsg{Devices: nil, Err: fmt.Errorf("player not initialized")}
		}
		devs, err := m.player.GetDevices(context.Background())
		return devices.DevicesLoadedMsg{Devices: devs, Err: err}
	}
}

func (m *Model) pollPlayerStateCmd() tea.Cmd {
	return func() tea.Msg {
		if utils.IsYouTubeMusicMode() {
			if m.mpvPlayer != nil {
				return ytPlayerStateMsg{
					positionMs: m.mpvPlayer.PositionMs(),
					durationMs: m.mpvPlayer.DurationMs(),
					playing:    m.mpvPlayer.IsPlaying(),
				}
			}
			return nil
		}
		if m.player == nil {
			return playerStateMsg{state: nil, err: fmt.Errorf("player not initialized")}
		}
		if m.player.GetDeviceID() == "" {
			_, _ = m.player.AutoSelectLocalDevice(context.Background())
		}
		state, err := m.player.GetPlayerState(context.Background())
		return playerStateMsg{state: state, err: err}
	}
}

func (m *Model) waitForMpvTrackEndCmd() tea.Cmd {
	return func() tea.Msg {
		if m.mpvPlayer == nil {
			return nil
		}
		endCh := m.mpvPlayer.EndChannel()
		if endCh == nil {
			return nil
		}
		<-endCh
		return ytTrackEndedMsg{}
	}
}

func (m *Model) seedRadioCmd(videoID string, currentTrack ytmusic.Track) tea.Cmd {
	return func() tea.Msg {
		client := ytmusic.GetClient()
		tracks, err := client.GetRadioTracks(context.Background(), videoID, 25)
		if err != nil {
			logger.Log.Error().Err(err).Str("videoID", videoID).Msg("failed to seed radio tracks")
			return ytRadioLoadedMsg{seedVideoID: videoID, tracks: nil, err: err}
		}
		var fullQueue []ytmusic.Track
		if len(tracks) > 0 && tracks[0].VideoID == videoID {
			fullQueue = tracks
			if currentTrack.Title != "" {
				fullQueue[0].Title = currentTrack.Title
			}
			if currentTrack.Artist != "" {
				fullQueue[0].Artist = currentTrack.Artist
			}
			if currentTrack.ArtURL != "" {
				fullQueue[0].ArtURL = currentTrack.ArtURL
			}
		} else {
			fullQueue = append([]ytmusic.Track{currentTrack}, tracks...)
		}
		logger.Log.Info().Int("queueLen", len(fullQueue)).Str("videoID", videoID).Msg("radio queue loaded")
		return ytRadioLoadedMsg{seedVideoID: videoID, tracks: fullQueue, err: nil}
	}
}

func (m *Model) extendRadioCmd(videoID string) tea.Cmd {
	return func() tea.Msg {
		client := ytmusic.GetClient()
		tracks, err := client.GetRadioTracks(context.Background(), videoID, 25)
		if err != nil {
			return nil
		}
		return ytRadioExtendedMsg{tracks: tracks}
	}
}

func (m *Model) updateQueueDisplay() {
	if len(m.ytQueue) == 0 {
		return
	}
	current := m.ytQueue[m.ytQueueIndex]
	var nextTracks []ytmusic.Track
	if m.ytQueueIndex+1 < len(m.ytQueue) {
		nextTracks = m.ytQueue[m.ytQueueIndex+1:]
	}

	q := &spotapi.Queue{
		CurrentlyPlaying: spotapi.FullTrack{
			SimpleTrack: spotapi.SimpleTrack{
				Name: current.Title,
				Artists: []spotapi.SimpleArtist{{Name: current.Artist}},
				URI: spotapi.URI(fmt.Sprintf("ytmusic:%s|%s|%s|%s", current.VideoID, current.Title, current.Artist, current.ArtURL)),
				Duration: spotapi.Numeric(current.DurationMs),
			},
		},
	}
	for _, t := range nextTracks {
		q.Items = append(q.Items, spotapi.FullTrack{
			SimpleTrack: spotapi.SimpleTrack{
				Name: t.Title,
				Artists: []spotapi.SimpleArtist{{Name: t.Artist}},
				URI: spotapi.URI(fmt.Sprintf("ytmusic:%s|%s|%s|%s", t.VideoID, t.Title, t.Artist, t.ArtURL)),
				Duration: spotapi.Numeric(t.DurationMs),
			},
		})
	}
	m.mediaCenter.SetQueue(q)
}

func (m *Model) playYtTrackCmd(t ytmusic.Track) tea.Cmd {
	videoID := t.VideoID
	title := t.Title
	artist := t.Artist
	artURL := t.ArtURL

	m.lastLyricsTrack = title
	m.lastLyricsArtist = artist
	m.songInfo = common.SongInfo{
		Title:    title,
		Artist:   artist,
		Position: 0,
		Duration: t.DurationMs,
	}
	m.mediaCenter.SetDisplayFromSong(m.songInfo)
	m.updatePlayerStatus()

	streamURL := ytmusic.GetClient().GetStreamURL(videoID)
	if m.mpvPlayer != nil {
		_ = m.mpvPlayer.Play(streamURL)
	}
	m.playing = true
	m.playerReady = true

	var cmds []tea.Cmd
	cmds = append(cmds, m.fetchLyricsCmd(title, artist))
	if artURL != "" {
		artCols, artRows := m.desiredArtworkDimensions()
		m.lastArtworkURL = artURL
		cmds = append(cmds, m.fetchArtworkCmd(artURL, artCols, artRows))
	}
	cmds = append(cmds, m.waitForMpvTrackEndCmd())
	return tea.Batch(cmds...)
}

func (m *Model) playNextYtTrackCmd() tea.Cmd {
	if len(m.ytQueue) == 0 || m.ytQueueIndex >= len(m.ytQueue)-1 {
		return nil
	}
	m.ytQueueIndex++
	track := m.ytQueue[m.ytQueueIndex]
	m.updateQueueDisplay()

	var cmds []tea.Cmd
	cmds = append(cmds, m.playYtTrackCmd(track))
	if len(m.ytQueue)-m.ytQueueIndex <= 3 {
		cmds = append(cmds, m.extendRadioCmd(track.VideoID))
	}
	return tea.Batch(cmds...)
}

func (m *Model) playPrevYtTrackCmd() tea.Cmd {
	if m.songInfo.Position > 3000 && m.mpvPlayer != nil {
		_ = m.mpvPlayer.Seek(0)
		m.songInfo.Position = 0
		m.updatePlayerStatus()
		return nil
	}
	if m.ytQueueIndex <= 0 || len(m.ytQueue) == 0 {
		if m.mpvPlayer != nil {
			_ = m.mpvPlayer.Seek(0)
		}
		return nil
	}
	m.ytQueueIndex--
	track := m.ytQueue[m.ytQueueIndex]
	m.updateQueueDisplay()
	return m.playYtTrackCmd(track)
}

func (m *Model) fetchLyricsCmd(track, artist string) tea.Cmd {
	return func() tea.Msg {
		svc := corelyrics.GetService()
		l, err := svc.FetchLyrics(track, artist)
		return lyricsLoadedMsg{
			track:  track,
			artist: artist,
			lyrics: l,
			err:    err,
		}
	}
}

func (m *Model) fetchQueueCmd() tea.Cmd {
	return func() tea.Msg {
		if m.player == nil {
			return queueLoadedMsg{err: fmt.Errorf("player not initialized")}
		}
		q, err := m.player.GetQueue(context.Background())
		return queueLoadedMsg{queue: q, err: err}
	}
}

func (m *Model) desiredArtworkDimensions() (cols, rows int) {
	h := m.height
	if h <= 0 {
		return 36, 18
	}
	// Total cassette + header is roughly ~24 rows
	avail := h - 24
	if avail < 10 {
		avail = 10
	} else if avail > 22 {
		avail = 22
	}
	rows = avail - 2
	if rows < 8 {
		rows = 8
	} else if rows > 20 {
		rows = 20
	}
	cols = rows * 2
	return cols, rows
}

func (m *Model) fetchArtworkCmd(imageURL string, cols, rows int) tea.Cmd {
	return func() tea.Msg {
		ansi, err := artwork.GetRenderer().Render(context.Background(), imageURL, cols, rows)
		return artworkLoadedMsg{
			imageURL: imageURL,
			ansi:     ansi,
			err:      err,
		}
	}
}

func (m *Model) quitAfterFatalError() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return fatalQuitMsg{}
	}
}

func (m *Model) setFatalError(err error) tea.Cmd {
	if err == nil {
		return nil
	}
	m.fatalErr = err
	logger.Log.Error().Err(err).Msg("fatal application error")
	return m.quitAfterFatalError()
}

func (m *Model) showActionError(action string, err error) {
	if err == nil {
		return
	}
	if coreplayer.IsNoActiveDeviceError(err) {
		m.mediaCenter.SetDisplay("No Active Device Found")
		m.playerReady = false
		m.playing = false
		m.updatePlayerStatus()
		return
	}
	if action == "" {
		action = "Error"
	}
	m.mediaCenter.SetDisplay(fmt.Sprintf("%s: %v", action, err))
}

func (m *Model) markVolumeOverlay() {
	m.volumeOverlayUntil = time.Now().Add(1500 * time.Millisecond)
}

func (m *Model) previewVolume(delta int) {
	maxVolume := m.volumeInfo.Max
	if maxVolume <= 0 {
		maxVolume = 100
	}
	m.volumeInfo.Volume = max(0, min(maxVolume, m.volumeInfo.Volume+delta))
}

func (m *Model) advancePlayback(elapsedMs int) {
	if !m.playing || elapsedMs <= 0 || m.songInfo.Duration <= 0 {
		return
	}
	m.songInfo.Position = min(m.songInfo.Position+elapsedMs, m.songInfo.Duration)
	m.updatePlayerStatus()
}

func (m *Model) updatePlayerStatus() {
	maxVolume := m.volumeInfo.Max
	if maxVolume <= 0 {
		maxVolume = 100
	}
	shuffled := false
	if m.player != nil {
		shuffled = m.player.Shuffled()
	}
	m.mediaCenter.UpdatePlayerStatus(player.Status{
		PlayerReady: m.playerReady,
		Playing:     m.playing,
		Position:    m.songInfo.Position,
		Duration:    m.songInfo.Duration,
		Volume:      m.volumeInfo.Volume,
		MaxVolume:   maxVolume,
		Shuffled:    shuffled,
		TrackName:   m.songInfo.Title,
		ArtistName:  m.songInfo.Artist,
	})
	m.mediaCenter.SetNowPlayingSong(m.songInfo)
	m.mediaCenter.SetNowPlayingVolume(m.volumeInfo)
	devName := "cassette (PC)"
	if m.player != nil && m.player.GetDeviceName() != "" {
		devName = m.player.GetDeviceName()
	}
	m.mediaCenter.SetNowPlayingStatus(m.playing, devName, shuffled)
	m.updateMprisState()
}

func (m *Model) playPause() error {
	if utils.IsYouTubeMusicMode() && m.mpvPlayer != nil {
		if m.playing {
			err := m.mpvPlayer.Pause()
			if err == nil {
				m.playing = false
			}
			return err
		}
		err := m.mpvPlayer.Resume()
		if err == nil {
			m.playing = true
		}
		return err
	}
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.PlayPause(context.Background(), m.playing)
}

func (m *Model) seekForward() error {
	step := utils.GetConfig().Player.SeekStepMs
	if step <= 0 {
		step = 5000
	}
	if utils.IsYouTubeMusicMode() && m.mpvPlayer != nil {
		return m.mpvPlayer.SeekRelative(step)
	}
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Seek(context.Background(), step, true, m.songInfo.Position)
}

func (m *Model) seekBackward() error {
	step := utils.GetConfig().Player.SeekStepMs
	if step <= 0 {
		step = 5000
	}
	if utils.IsYouTubeMusicMode() && m.mpvPlayer != nil {
		return m.mpvPlayer.SeekRelative(-step)
	}
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Seek(context.Background(), -step, true, m.songInfo.Position)
}

func (m *Model) next() error {
	if utils.IsYouTubeMusicMode() {
		return nil
	}
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Next(context.Background())
}

func (m *Model) previous() error {
	if utils.IsYouTubeMusicMode() {
		return nil
	}
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Previous(context.Background())
}

func (m *Model) shuffle(shuffle bool) error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Shuffle(context.Background(), shuffle)
}

func (m *Model) changeVolume(deltaPercent int) (common.VolumeInfo, error) {
	if utils.IsYouTubeMusicMode() && m.mpvPlayer != nil {
		target := max(0, min(100, m.volumeInfo.Volume+deltaPercent))
		if err := m.mpvPlayer.SetVolume(target); err != nil {
			return common.VolumeInfo{}, err
		}
		m.volumeInfo = common.VolumeInfo{Volume: target, Max: 100}
		return m.volumeInfo, nil
	}
	if m.player == nil {
		return common.VolumeInfo{}, fmt.Errorf("player not ready")
	}

	target := max(0, min(100, m.volumeInfo.Volume+deltaPercent))
	if err := m.player.SetVolume(context.Background(), target); err != nil {
		return common.VolumeInfo{}, err
	}
	return common.VolumeInfo{Volume: target, Max: 100}, nil
}

func (m *Model) playPauseCmd() tea.Cmd {
	targetPlaying := !m.playing
	return func() tea.Msg {
		if err := m.playPause(); err != nil {
			return transportErrMsg{err: err, action: "Failed to play/pause track"}
		}
		return playPauseOkMsg{playing: targetPlaying}
	}
}

func (m *Model) seekForwardCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.seekForward(); err != nil {
			return transportErrMsg{err: err, action: "Failed to seek forward"}
		}
		return nil
	}
}

func (m *Model) seekBackwardCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.seekBackward(); err != nil {
			return transportErrMsg{err: err, action: "Failed to seek backward"}
		}
		return nil
	}
}

func (m *Model) nextCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.next(); err != nil {
			return transportErrMsg{err: err, action: "Failed to skip to next track"}
		}
		return nextTrackOkMsg{}
	}
}

func (m *Model) previousCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.previous(); err != nil {
			return transportErrMsg{err: err, action: "Failed to skip to previous track"}
		}
		return prevTrackOkMsg{}
	}
}

func (m *Model) shuffleCmd() tea.Cmd {
	targetShuffle := true
	if m.player != nil {
		targetShuffle = !m.player.Shuffled()
	}
	return func() tea.Msg {
		if err := m.shuffle(targetShuffle); err != nil {
			return transportErrMsg{err: err, action: "Failed to toggle shuffle"}
		}
		return shuffleOkMsg{shuffled: targetShuffle}
	}
}

func (m *Model) incrementVolumeCmd() tea.Cmd {
	return func() tea.Msg {
		step := utils.GetConfig().Player.VolumeStep
		if step <= 0 {
			step = 5
		}
		volumeInfo, err := m.changeVolume(step)
		if err != nil {
			return transportErrMsg{err: err, action: "Failed to increase volume"}
		}
		return volumeChangedMsg{volumeInfo: volumeInfo}
	}
}

func (m *Model) decrementVolumeCmd() tea.Cmd {
	return func() tea.Msg {
		step := utils.GetConfig().Player.VolumeStep
		if step <= 0 {
			step = 5
		}
		volumeInfo, err := m.changeVolume(-step)
		if err != nil {
			return transportErrMsg{err: err, action: "Failed to decrease volume"}
		}
		return volumeChangedMsg{volumeInfo: volumeInfo}
	}
}

func decodeOffsetCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	value, err := strconv.Atoi(cursor)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func encodeOffsetCursor(offset int) string {
	if offset < 0 {
		offset = 0
	}
	return strconv.Itoa(offset)
}

func totalPages(totalItems int, pageSize int) int {
	if totalItems <= 0 || pageSize <= 0 {
		return 1
	}
	pages := totalItems / pageSize
	if totalItems%pageSize != 0 {
		pages++
	}
	if pages <= 0 {
		return 1
	}
	return pages
}

func paginationFromOffset(offset, count, total, pageSize int) common.PaginationInfo {
	currentPage := 1
	if pageSize > 0 {
		currentPage = (offset / pageSize) + 1
	}
	hasNext := offset+count < total
	nextCursor := ""
	if hasNext {
		nextCursor = encodeOffsetCursor(offset + pageSize)
	}
	return common.PaginationInfo{
		CurrentPage: currentPage,
		TotalPages:  totalPages(total, pageSize),
		TotalItems:  total,
		HasNext:     hasNext,
		NextCursor:  nextCursor,
	}
}

func paginationFromCursor(page, count, total, pageSize int, nextCursor string) common.PaginationInfo {
	hasNext := nextCursor != "" && count > 0
	if page <= 0 {
		page = 1
	}
	return common.PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages(total, pageSize),
		TotalItems:  total,
		HasNext:     hasNext,
		NextCursor:  nextCursor,
	}
}

func calcVolumeDelta(maxVolume, stepPercent int) int {
	return maxVolume * stepPercent / 100
}

func ExitIfRunFails(err error) {
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to run program")
		os.Exit(1)
	}
}

func IsZenMode() bool {
	return os.Getenv("ZEN_MODE") != ""
}

func (m *Model) updateMprisState() {
	if m.mprisServer == nil {
		return
	}
	shuffled := false
	if m.player != nil {
		shuffled = m.player.Shuffled()
	}
	m.mprisServer.UpdatePlayback(mpris.PlaybackState{
		Playing:    m.playing,
		Title:      m.songInfo.Title,
		Artist:     m.songInfo.Artist,
		Album:      m.songInfo.Album,
		ArtURL:     m.lastArtworkURL,
		TrackID:    m.lastTrackID,
		PositionMs: m.songInfo.Position,
		DurationMs: m.songInfo.Duration,
		Volume:     m.volumeInfo.Volume,
		Shuffled:   shuffled,
	})
}

func (m *Model) initMpris() {
	if m.mprisServer != nil {
		return
	}

	cb := mpris.Callbacks{
		OnPlayPause: func() error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			err := m.playPause()
			if m.program != nil {
				m.program.Send(mprisPollStateMsg{})
			}
			return err
		},
		OnPlay: func() error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			if !m.playing {
				err := m.playPause()
				if m.program != nil {
					m.program.Send(mprisPollStateMsg{})
				}
				return err
			}
			return nil
		},
		OnPause: func() error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			if m.playing {
				err := m.playPause()
				if m.program != nil {
					m.program.Send(mprisPollStateMsg{})
				}
				return err
			}
			return nil
		},
		OnNext: func() error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			err := m.next()
			if m.program != nil {
				m.program.Send(mprisPollStateMsg{})
			}
			return err
		},
		OnPrevious: func() error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			err := m.previous()
			if m.program != nil {
				m.program.Send(mprisPollStateMsg{})
			}
			return err
		},
		OnStop: func() error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			if m.playing {
				err := m.playPause()
				if m.program != nil {
					m.program.Send(mprisPollStateMsg{})
				}
				return err
			}
			return nil
		},
		OnSeek: func(offsetUs int64) error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			offsetMs := int(offsetUs / 1000)
			err := m.player.Seek(context.Background(), offsetMs, true, m.songInfo.Position)
			if m.program != nil {
				m.program.Send(mprisPollStateMsg{})
			}
			return err
		},
		OnSetPosition: func(trackID string, positionUs int64) error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			posMs := int(positionUs / 1000)
			err := m.player.Seek(context.Background(), posMs, false, 0)
			if m.program != nil {
				m.program.Send(mprisPollStateMsg{})
			}
			return err
		},
		OnSetVolume: func(volumePercent int) error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			err := m.player.SetVolume(context.Background(), volumePercent)
			if m.program != nil {
				m.program.Send(volumeChangedMsg{
					volumeInfo: common.VolumeInfo{
						Volume: volumePercent,
						Max:    100,
					},
				})
			}
			return err
		},
		OnSetShuffle: func(shuffle bool) error {
			if m.player == nil {
				return fmt.Errorf("player not initialized")
			}
			err := m.shuffle(shuffle)
			if m.program != nil {
				m.program.Send(shuffleOkMsg{shuffled: shuffle})
			}
			return err
		},
		OnQuit: func() error {
			if m.program != nil {
				m.program.Quit()
			}
			return nil
		},
	}

	srv, err := mpris.NewServer(cb)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("failed to initialize MPRIS D-Bus server")
		return
	}
	m.mprisServer = srv
	m.updateMprisState()
}

// --- Welcome Popup (one-time changelog + crypto donation) ---

const welcomeVersion = "2.0.0"

// WelcomeModel is a one-time modal popup showing the changelog and donation info.
type WelcomeModel struct {
	open bool
}

func (w *WelcomeModel) IsOpen() bool {
	return w != nil && w.open
}

func (w *WelcomeModel) Close() {
	if w != nil {
		w.open = false
	}
}

func (w *WelcomeModel) View() string {
	if w == nil || !w.open {
		return ""
	}
	th := theme.Get()
	accent := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	cryptoVal := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	innerWidth := 70
	divider := lipgloss.NewStyle().Foreground(th.BorderColor()).Render(strings.Repeat("─", innerWidth))

	title := accent.Render("CASSETTE OVERHAUL — WHAT'S NEW")
	titleLine := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(title)

	lines := []string{
		titleLine,
		"",
		bright.Render("  🎵 YouTube Music First"),
		dim.Render("     Free streaming, no Spotify Premium required."),
		bright.Render("  🎙️ Word-Synced Karaoke Lyrics"),
		dim.Render("     Multi-source word-by-word highlighting (LRCLIB, Portato, etc)."),
		bright.Render("  ⚡ Zero-CPU Background"),
		dim.Render("     Animations & artwork pause when terminal is idle/unfocused."),
		bright.Render("  🚀 Fixed Boot Autostart"),
		dim.Render("     Reliable startup wrapper in KDE Plasma / GNOME / Wayland."),
		bright.Render("  🔀 Dual-Source Engine"),
		dim.Render("     YouTube Music default + Spotify mode for exclusive tracks."),
		"",
		divider,
		"",
		accent.Render("  Cassette is 100% free & open source."),
		dim.Render("  No telemetry, ads, or pro tiers. Ever."),
		"",
		bright.Render("  Support development via crypto donations:"),
		dim.Render("  • Nano (XNO):"),
		cryptoVal.Render("    nano_1zqdw3qf1z8k3jx8jintaiwpo3yz7zqh1me4ph5j439ts8hsppx8dzy4xcsz"),
		dim.Render("  • USDC (Base):"),
		cryptoVal.Render("    0x3f262ee685ced4a8270cece45ebdfdb2b18f54b5"),
		dim.Render("  • Zcash (ZEC):"),
		cryptoVal.Render("    u1z9k30yyvy63f5w0jypt02kvvw6dcpcgprlhzmsgc57mw6qc8rtuc5tfd9ny4at"),
		cryptoVal.Render("    qhr448udexhkuc8xgl0z5rd9njuxnwl7kh2ahqwcqlydt9dpr4t40eawr5st74as"),
		cryptoVal.Render("    5jed669993epsnwuejnrwv4yrkx065pqvmt0cr8gdwggcv2djp"),
		"",
		lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(accent.Render("[Press Any Key to dismiss]")),
	}

	body := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.BorderColor()).
		Padding(0, 1).
		Width(innerWidth + 4)

	return boxStyle.Render(body)
}

func (m *Model) initWelcomePopup() {
	cfg := utils.GetConfig()
	if cfg.SeenWelcomeVersion >= welcomeVersion {
		return
	}
	m.welcomeModel = &WelcomeModel{open: true}
}

func (m *Model) dismissWelcome() {
	if m.welcomeModel != nil {
		m.welcomeModel.Close()
	}
	go func() {
		_ = utils.SetSeenWelcomeVersion(welcomeVersion)
	}()
}

// backgroundPlaceholderView renders a minimal view when the terminal is unfocused,
// avoiding all expensive ANSI art, album art, and animation rendering.
func (m *Model) backgroundPlaceholderView() string {
	th := theme.Get()
	accent := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	status := "paused"
	if m.playing {
		status = "playing"
	}

	info := ""
	if m.songInfo.Title != "" {
		info = fmt.Sprintf("%s — %s", m.songInfo.Title, m.songInfo.Artist)
	}

	line1 := accent.Render("♪ cassette") + dim.Render(" • [background] • CPU 0%")
	line2 := dim.Render(fmt.Sprintf("  %s", status))
	line3 := ""
	if info != "" {
		line3 = dim.Render("  " + info)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, line1, line2, line3)
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}
