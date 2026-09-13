package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"cassette/core/artwork"
	"cassette/core/auth"
	"cassette/core/logger"
	corelyrics "cassette/core/lyrics"
	coreplayer "cassette/core/player"
	"cassette/core/ticker"
	"cassette/core/utils"
	"cassette/spotify"
	uiauth "cassette/ui/v1/auth"
	"cassette/ui/v1/common"
	"cassette/ui/v1/devices"
	"cassette/ui/v1/mediacenter"
	"cassette/ui/v1/player"
	spotapi "github.com/zmb3/spotify/v2"
)

type Model struct {
	authModel          *uiauth.Model
	devicesModel       *devices.Model
	devicePickerOpen   bool
	playing            bool
	playerReady        bool
	songInfo           common.SongInfo
	volumeInfo         common.VolumeInfo
	volumeOverlayUntil time.Time
	fatalErr           error
	player             *coreplayer.Player
	localDaemon        *coreplayer.LocalDaemon
	spotifyClient      *spotify.SpotifyClient
	mediaCenter        mediacenter.Model
	lastLyricsTrack    string
	lastLyricsArtist   string
	lastArtworkURL     string
	width              int
	height             int
	help               help.Model
	keys               common.AppKeyMap
	requestHandlers    map[common.MediaRequestKind]func(common.MediaRequest) tea.Cmd
}

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

type playerStateMsg struct {
	state *spotapi.PlayerState
	err   error
}

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

type shuffleOkMsg struct{}

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
		authModel:    uiauth.NewModel(),
		devicesModel: devices.NewModel(false),
		mediaCenter:  mediacenter.NewModel(keys),
		help:         newHelpModel(),
		keys:         keys,
		volumeInfo:   common.VolumeInfo{Volume: 50, Max: 100},
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
	_, err := tea.NewProgram(model).Run()
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
}

func (m *Model) shutdown() {
	if m.localDaemon != nil {
		m.localDaemon.Stop()
	}
}

func (m *Model) start() error {
	ctx := context.Background()
	var err error
	m.authModel = uiauth.NewModel()
	if m.width != 0 || m.height != 0 {
		m.authModel.SetSize(m.width, m.height)
		m.devicesModel.SetSize(m.width, m.height)
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
}

func (m *Model) playPause() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.PlayPause(context.Background(), m.playing)
}

func (m *Model) seekForward() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	step := utils.GetConfig().Player.SeekStepMs
	if step <= 0 {
		step = 5000
	}
	return m.player.Seek(context.Background(), step, true, m.songInfo.Position)
}

func (m *Model) seekBackward() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	step := utils.GetConfig().Player.SeekStepMs
	if step <= 0 {
		step = 5000
	}
	return m.player.Seek(context.Background(), -step, true, m.songInfo.Position)
}

func (m *Model) next() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Next(context.Background())
}

func (m *Model) previous() error {
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
		return nil
	}
}

func (m *Model) previousCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.previous(); err != nil {
			return transportErrMsg{err: err, action: "Failed to skip to previous track"}
		}
		return nil
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
		return shuffleOkMsg{}
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
