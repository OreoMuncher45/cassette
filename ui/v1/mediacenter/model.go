package mediacenter

import (
	tea "charm.land/bubbletea/v2"
	corelyrics "cassette/core/lyrics"
	"cassette/ui/v1/common"
	"cassette/ui/v1/displayscreen"
	"cassette/ui/v1/lyrics"
	"cassette/ui/v1/mediapanel"
	"cassette/ui/v1/nowplaying"
	"cassette/ui/v1/player"
	"cassette/ui/v1/queue"
	spotapi "github.com/zmb3/spotify/v2"
)

type LeftPanelMode int

const (
	LeftPanelLyrics LeftPanelMode = iota
	LeftPanelLibrary
	LeftPanelArtwork
	LeftPanelClosed
)

type Model struct {
	mediaPanel    mediapanel.Model
	player        player.Model
	displayScreen displayscreen.Model
	lyricsModel   lyrics.Model
	queueModel    queue.Model
	nowPlaying    nowplaying.Model
	artANSI       string
	currentSong   common.SongInfo
	leftPanel     LeftPanelMode
	queueOpen     bool
	zenMode       bool
	keys          common.AppKeyMap
}

func NewModel(keys common.AppKeyMap) Model {
	return Model{
		mediaPanel:    mediapanel.NewModel(keys),
		player:        player.NewModel(),
		displayScreen: displayscreen.NewModel(),
		lyricsModel:   lyrics.NewModel(),
		queueModel:    queue.NewModel(),
		nowPlaying:    nowplaying.NewModel(),
		leftPanel:     LeftPanelLyrics, // Lyrics on the left by default
		queueOpen:     true,            // Queue on the right by default
		keys:          keys,
	}
}

func (m *Model) SetDisplay(text string) {
	m.displayScreen.SetDisplay(text)
}

func (m *Model) SetDisplayFromSong(song common.SongInfo) {
	m.displayScreen.SetDisplayFromSong(song)
}

func (m *Model) UpdatePlayerStatus(status player.Status) {
	m.player.UpdateStatus(status)
}

func (m *Model) TickPlayer(playing bool) {
	m.player.NextFrame(playing)
}

func (m *Model) TickDisplay() tea.Cmd {
	return m.displayScreen.NextFrame()
}

func (m *Model) TickButtons() {
	m.player.NextButtonFrame()
}

func (m *Model) PressButton(kind player.ButtonKind) tea.Cmd {
	return m.player.HandleButtonPress(kind)
}

func (m *Model) ShowVolume() tea.Cmd {
	return m.player.ShowVolume()
}

func (m *Model) HideVolume() {
	m.player.HideVolume()
}

func (m *Model) StartLoading(kind common.ListKind) tea.Cmd {
	return m.mediaPanel.StartLoading(kind)
}

func (m *Model) SetContent(entities []common.Entity, kind common.ListKind, pagination common.PaginationInfo, request common.MediaRequest) tea.Cmd {
	return m.mediaPanel.SetContent(entities, kind, pagination, request)
}

func (m *Model) SetStatus(kind common.ListKind, message string) tea.Cmd {
	return m.mediaPanel.SetStatus(kind, message)
}

func (m *Model) SetLyrics(l *corelyrics.Lyrics) {
	m.lyricsModel.SetLyrics(l)
}

func (m *Model) SetLyricsPosition(posMs int) {
	m.lyricsModel.SetPosition(posMs)
}

func (m *Model) SetLyricsTrack(track, artist string) {
	m.lyricsModel.SetTrack(track, artist)
}

func (m *Model) SetLyricsLoading(loading bool) {
	m.lyricsModel.SetLoading(loading)
}

func (m *Model) SetQueue(q *spotapi.Queue) {
	m.queueModel.SetQueue(q)
}

func (m *Model) LeftPanel() LeftPanelMode {
	return m.leftPanel
}

func (m *Model) IsQueueOpen() bool {
	return m.queueOpen
}

func (m *Model) ToggleLyrics() {
	if m.leftPanel == LeftPanelLyrics {
		m.leftPanel = LeftPanelClosed
	} else {
		m.leftPanel = LeftPanelLyrics
		m.mediaPanel.CloseInfo()
	}
}

func (m *Model) ToggleArtwork() {
	if m.leftPanel == LeftPanelArtwork {
		m.leftPanel = LeftPanelLyrics
	} else {
		m.leftPanel = LeftPanelArtwork
		m.mediaPanel.CloseInfo()
	}
}

func (m *Model) ToggleQueue() {
	m.queueOpen = !m.queueOpen
}

func (m *Model) SwapLeftPanel() {
	if m.leftPanel == LeftPanelLyrics {
		m.leftPanel = LeftPanelLibrary
	} else if m.leftPanel == LeftPanelLibrary {
		m.leftPanel = LeftPanelLyrics
		m.mediaPanel.CloseInfo()
	} else {
		m.leftPanel = LeftPanelLyrics
	}
}

func (m *Model) ToggleLibrary() {
	if m.leftPanel == LeftPanelLibrary {
		m.leftPanel = LeftPanelLyrics
		m.mediaPanel.CloseInfo()
	} else {
		m.leftPanel = LeftPanelLibrary
	}
}

func (m *Model) CloseLibrary() {
	if m.leftPanel == LeftPanelLibrary {
		m.leftPanel = LeftPanelLyrics
	}
	m.mediaPanel.CloseInfo()
}

func (m *Model) IsOpen() bool {
	return m.leftPanel == LeftPanelLibrary
}

func (m *Model) CycleLibraryNext() tea.Cmd {
	return m.mediaPanel.ActivateNextPanel()
}

func (m *Model) CycleLibraryPrev() tea.Cmd {
	return m.mediaPanel.ActivatePrevPanel()
}

func (m *Model) LibraryDepth() int {
	return m.mediaPanel.Depth()
}

func (m *Model) InfoOpen() bool {
	return m.mediaPanel.InfoOpen()
}

func (m *Model) IsZenMode() bool {
	return m.zenMode
}

func (m *Model) SearchFocused() bool {
	return m.mediaPanel.SearchFocused()
}

func (m *Model) SetArtwork(ansi string) {
	m.artANSI = ansi
	m.nowPlaying.SetArtwork(ansi)
}

func (m *Model) SetNowPlayingSong(song common.SongInfo) {
	m.currentSong = song
	m.nowPlaying.SetSong(song)
}

func (m *Model) SetNowPlayingStatus(playing bool, device string, shuffled bool) {
	m.nowPlaying.SetStatus(playing, device, shuffled)
}

func (m *Model) SetNowPlayingVolume(v common.VolumeInfo) {
	m.nowPlaying.SetVolume(v)
}
