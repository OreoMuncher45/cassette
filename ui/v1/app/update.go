package app

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"cassette/core/logger"
	coreplayer "cassette/core/player"
	"cassette/core/ticker"
	uiauth "cassette/ui/v1/auth"
	"cassette/ui/v1/common"
	"cassette/ui/v1/devices"
	"cassette/ui/v1/player"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If search input is focused, route keystrokes directly to mediaCenter and do not trigger global shortcuts
	if m.mediaCenter.SearchFocused() {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if cmd, handled := m.handleSystemMessages(msg); handled {
			return m, cmd
		}
		centerCmd := m.mediaCenter.Update(msg)
		return m, centerCmd
	}

	if cmd, handled := m.handleShellInput(msg); handled {
		return m, cmd
	}

	if m.authModel != nil && m.authModel.State() < uiauth.Authenticated {
		newModel, cmd := m.authModel.Update(msg)
		m.authModel = newModel.(*uiauth.Model)
		return m, cmd
	}

	// If device selection screen is active, route input to it
	if m.devicePickerOpen && m.devicesModel != nil {
		if devMsg, ok := msg.(tea.KeyPressMsg); ok && devMsg.String() == "r" {
			m.devicesModel.SetLoading()
			return m, m.fetchDevicesCmd()
		}
		newDevModel, devCmd := m.devicesModel.Update(msg)
		m.devicesModel = newDevModel
		if cmd, handled := m.handleSystemMessages(msg); handled {
			return m, tea.Batch(devCmd, cmd)
		}
		return m, devCmd
	}

	if cmd, handled := m.handleSystemMessages(msg); handled {
		return m, cmd
	}
	if m.fatalErr != nil {
		return m, nil
	}
	centerCmd := m.mediaCenter.Update(msg)

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(keyMsg, m.keys.ToggleQueue) && m.mediaCenter.IsQueueOpen() {
			centerCmd = tea.Batch(centerCmd, m.fetchQueueCmd())
		}
	}

	if m.mediaCenter.IsOpen() {
		return m, centerCmd
	}

	if cmd, handled := m.handleTransportInput(msg, centerCmd); handled {
		return m, cmd
	}
	return m, centerCmd
}

func (m *Model) handleShellInput(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.ToggleHelp):
			m.help.ShowAll = !m.help.ShowAll
			return nil, true
		case key.Matches(msg, m.keys.Quit):
			return tea.Quit, true
		case key.Matches(msg, m.keys.Devices):
			if m.authModel != nil && m.authModel.State() == uiauth.Authenticated {
				m.devicePickerOpen = !m.devicePickerOpen
				if m.devicePickerOpen {
					m.devicesModel.SetLoading()
					m.devicesModel.SetCanDismiss(true)
					return m.fetchDevicesCmd(), true
				}
				return nil, true
			}
		}
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		if m.lastArtworkURL != "" {
			artCols, artRows := m.desiredArtworkDimensions()
			if artCols != m.lastArtCols || artRows != m.lastArtRows {
				m.lastArtCols = artCols
				m.lastArtRows = artRows
				return m.fetchArtworkCmd(m.lastArtworkURL, artCols, artRows), true
			}
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) handleSystemMessages(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case fatalErrMsg:
		return m.setFatalError(msg.err), true
	case fatalQuitMsg:
		return tea.Quit, true
	case uiauth.State:
		if msg == uiauth.Authenticated {
			logger.Log.Info().Msg("authenticated with spotify")
			return m.Init(), true
		}
	case devices.DevicesLoadedMsg:
		if m.devicesModel != nil {
			m.devicesModel.Update(msg)
		}
		return nil, true
	case devices.DeviceSelectedMsg:
		m.devicePickerOpen = false
		m.playerReady = true
		if m.devicesModel != nil {
			m.devicesModel.SetCanDismiss(true)
		}
		if m.player != nil {
			m.player.SetDevice(msg.Device.ID, msg.Device.Name)
			_ = m.player.TransferPlayback(context.Background(), msg.Device.ID, false)
		}
		requestCmd := tea.Cmd(func() tea.Msg {
			return common.RootMediaRequestForListKind(common.Playlists, "")
		})
		return tea.Batch(requestCmd, m.pollPlayerStateCmd()), true
	case devices.DeviceDismissedMsg:
		m.devicePickerOpen = false
		return nil, true
	case ticker.TickFastMsg:
		m.advancePlayback(180)
		m.mediaCenter.TickPlayer(m.playing)
		m.mediaCenter.SetLyricsPosition(m.songInfo.Position)
		return ticker.DoTickFast(), true
	case ticker.TickMsg:
		displayCmd := m.mediaCenter.TickDisplay()
		if !m.devicePickerOpen && m.playerReady {
			return tea.Batch(displayCmd, m.pollPlayerStateCmd()), true
		}
		return displayCmd, true
	case ticker.TickMsgVolume:
		m.mediaCenter.HideVolume()
		return nil, true
	case ticker.TickClickMsg:
		m.mediaCenter.TickButtons()
		return nil, true
	case common.MediaRequest:
		var startCmd tea.Cmd
		logger.Log.Info().Int("kind", int(msg.Kind)).Str("cursor", msg.Cursor).Int("page", msg.Page).Msg("requesting media")
		if msg.ShowLoading {
			startCmd = m.mediaCenter.StartLoading(msg.PanelKind)
		}
		return tea.Batch(startCmd, m.handleMediaRequest(msg)), true
	case startupCompleteMsg:
		requestCmd := tea.Cmd(func() tea.Msg {
			return common.RootMediaRequestForListKind(common.Playlists, "")
		})
		return tea.Batch(requestCmd, m.pollPlayerStateCmd(), m.fetchQueueCmd()), true
	case playerStateMsg:
		if msg.err != nil {
			if coreplayer.IsNoActiveDeviceError(msg.err) {
				// Only reset display if we don't have any loaded song or known device
				if m.songInfo.Title == "" && (m.player == nil || m.player.GetDeviceID() == "") {
					m.playerReady = false
					m.playing = false
					m.mediaCenter.SetDisplay("No Active Device (press d)")
					m.updatePlayerStatus()
				} else {
					// We are simply paused or inactive on the current device
					m.playing = false
					m.updatePlayerStatus()
				}
			}
			return nil, true
		}
		if msg.state != nil {
			m.playerReady = true
			m.playing = msg.state.CurrentlyPlaying.Playing
			var extraCmds []tea.Cmd
			if msg.state.Item != nil {
				artist := joinArtists(msg.state.Item.Artists)
				track := msg.state.Item.Name
				m.songInfo = common.SongInfo{
					Title:    track,
					Artist:   artist,
					Album:    msg.state.Item.Album.Name,
					Position: int(msg.state.Progress),
					Duration: int(msg.state.Item.Duration),
				}
				m.mediaCenter.SetDisplayFromSong(m.songInfo)
				m.mediaCenter.SetLyricsPosition(int(msg.state.Progress))

				if track != m.lastLyricsTrack || artist != m.lastLyricsArtist {
					m.lastLyricsTrack = track
					m.lastLyricsArtist = artist
					m.mediaCenter.SetLyricsTrack(track, artist)
					extraCmds = append(extraCmds, m.fetchLyricsCmd(track, artist), m.fetchQueueCmd())
				}

				if len(msg.state.Item.Album.Images) > 0 {
					artURL := msg.state.Item.Album.Images[0].URL
					artCols, artRows := m.desiredArtworkDimensions()
					if artURL != m.lastArtworkURL || artCols != m.lastArtCols || artRows != m.lastArtRows {
						m.lastArtworkURL = artURL
						m.lastArtCols = artCols
						m.lastArtRows = artRows
						extraCmds = append(extraCmds, m.fetchArtworkCmd(artURL, artCols, artRows))
					}
				}
			}
			if msg.state.Device.Volume > 0 {
				m.volumeInfo.Volume = int(msg.state.Device.Volume)
				m.volumeInfo.Max = 100
			}
			m.updatePlayerStatus()
			if len(extraCmds) > 0 {
				return tea.Batch(extraCmds...), true
			}
		} else if m.songInfo.Title != "" {
			// Spotify returned 204 No Content (paused / idle)
			m.playing = false
			m.updatePlayerStatus()
		}
		return nil, true
	case lyricsLoadedMsg:
		if msg.track == m.lastLyricsTrack && msg.artist == m.lastLyricsArtist {
			if msg.err == nil && msg.lyrics != nil {
				m.mediaCenter.SetLyrics(msg.lyrics)
			} else {
				m.mediaCenter.SetLyrics(nil)
			}
		}
		return nil, true
	case queueLoadedMsg:
		if msg.err == nil && msg.queue != nil {
			m.mediaCenter.SetQueue(msg.queue)
		}
		return nil, true
	case artworkLoadedMsg:
		if msg.err == nil && msg.imageURL == m.lastArtworkURL {
			m.mediaCenter.SetArtwork(msg.ansi)
		}
		return nil, true
	case mediaLoadedMsg:
		return m.mediaCenter.SetContent(msg.entities, msg.kind, msg.pagination, msg.request), true
	case mediaLoadErrMsg:
		logger.Log.Error().Err(msg.err).Msg("failed to get user library")
		m.mediaCenter.StartLoading(msg.request.PanelKind)
		return m.mediaCenter.SetStatus(msg.request.PanelKind, "Failed to load library"), true
	case playTrackErrMsg:
		logger.Log.Error().Err(msg.err).Msg("failed to play track")
		if coreplayer.IsNoActiveDeviceError(msg.err) {
			m.playerReady = false
			m.playing = false
			m.mediaCenter.SetDisplay("No Active Device Found")
			m.updatePlayerStatus()
			return m.mediaCenter.SetStatus(msg.panelKind, "No active device (press d)"), true
		}
		m.mediaCenter.SetDisplay("Error: " + msg.err.Error())
		return m.mediaCenter.SetStatus(msg.panelKind, "Failed to play track"), true
	case playTrackOkMsg:
		m.playing = true
		m.playerReady = true
		m.updatePlayerStatus()
		return tea.Batch(m.mediaCenter.SetStatus(msg.panelKind, "Playing"), m.pollPlayerStateCmd()), true
	case playPauseOkMsg:
		m.playing = msg.playing
		m.updatePlayerStatus()
		return m.pollPlayerStateCmd(), true
	case volumeChangedMsg:
		m.volumeInfo = msg.volumeInfo
		m.markVolumeOverlay()
		m.updatePlayerStatus()
		return tea.Batch(m.mediaCenter.ShowVolume(), m.pollPlayerStateCmd()), true
	case shuffleOkMsg:
		m.updatePlayerStatus()
		return m.pollPlayerStateCmd(), true
	case transportErrMsg:
		logger.Log.Error().Err(msg.err).Str("action", msg.action).Msg("transport action failed")
		m.showActionError(msg.action, msg.err)
		return nil, true
	}
	return nil, false
}

func (m *Model) handleTransportInput(msg tea.Msg, centerCmd tea.Cmd) (tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}

	switch {
	case key.Matches(keyMsg, m.keys.PlayPause):
		button := player.PauseButton
		if !m.playing {
			button = player.PlayButton
		}
		return tea.Batch(m.mediaCenter.PressButton(button), m.playPauseCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.SeekForward):
		return tea.Batch(m.mediaCenter.PressButton(player.SeekForwardButton), m.seekForwardCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.SeekBackward):
		return tea.Batch(m.mediaCenter.PressButton(player.SeekBackwardButton), m.seekBackwardCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.NextTrack):
		return tea.Batch(m.mediaCenter.PressButton(player.NextButton), m.nextCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.PrevTrack):
		return tea.Batch(m.mediaCenter.PressButton(player.PreviousButton), m.previousCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.VolumeDown):
		return tea.Batch(m.decrementVolumeCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.VolumeUp):
		return tea.Batch(m.incrementVolumeCmd(), centerCmd), true
	case key.Matches(keyMsg, m.keys.Shuffle):
		return tea.Batch(m.shuffleCmd(), centerCmd), true
	}

	return nil, false
}

func (m *Model) handleMediaRequest(request common.MediaRequest) tea.Cmd {
	if request.Page <= 0 {
		request.Page = 1
	}
	handler, ok := m.requestHandlers[request.Kind]
	if !ok {
		return nil
	}
	return handler(request)
}
