package app

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"cassette/core/logger"
	"cassette/core/playlist"
	"cassette/core/utils"
	"cassette/core/ytmusic"
	"cassette/ui/v1/common"
	"github.com/zmb3/spotify/v2"
)

func (m *Model) handleGetUserPlaylists(request common.MediaRequest) tea.Cmd {
	if utils.IsYouTubeMusicMode() {
		return func() tea.Msg {
			var entities []common.Entity
			entities = append(entities,
				common.NewEntity("🔍 Search YouTube Music", "Press / to search songs, albums & artists", "ytmusic:search", ""),
			)

			// Load local playlists from core/playlist
			pls := playlist.GetPlaylists()
			for _, pl := range pls {
				desc := fmt.Sprintf("%d tracks • Local Playlist", len(pl.Tracks))
				uri := fmt.Sprintf("localpl:%s", pl.ID)
				art := ""
				if len(pl.Tracks) > 0 && pl.Tracks[0].ArtURL != "" {
					art = pl.Tracks[0].ArtURL
				}
				entities = append(entities, common.NewEntity(pl.Name, desc, uri, art))
			}

			pagination := paginationFromOffset(0, len(entities), len(entities), 20)
			return mediaLoadedMsg{entities: entities, kind: common.Playlists, pagination: pagination, request: request}
		}
	}
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.GetUserPlaylists(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyPlaylists(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Playlists, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetSavedTracks(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.GetSavedTracks(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifySavedTracks(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetSavedAlbums(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.GetSavedAlbums(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifySavedAlbums(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetFollowedArtists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		page, err := m.spotifyClient.GetFollowedArtists(context.Background(), request.Cursor)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtists(page)
		pagination := paginationFromCursor(request.Page, len(entities), int(page.Total), 10, page.Cursor.After)
		return mediaLoadedMsg{entities: entities, kind: common.Artists, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchPlaylists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchPlaylists(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyPlaylists(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Playlists, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchTracks(request common.MediaRequest) tea.Cmd {
	if utils.IsYouTubeMusicMode() {
		return func() tea.Msg {
			results, err := ytmusic.GetClient().Search(context.Background(), request.Query, 20)
			if err != nil {
				return mediaLoadErrMsg{err: err, request: request}
			}
			var entities []common.Entity
			for _, t := range results.Tracks {
				desc := t.Artist
				if t.Album != "" {
					desc += " • " + t.Album
				}
				id := fmt.Sprintf("ytmusic:%s|%s|%s|%s", t.VideoID, t.Title, t.Artist, t.ArtURL)
				entities = append(entities, common.NewEntity(t.Title, desc, id, t.ArtURL))
			}
			pagination := paginationFromOffset(0, len(entities), len(entities), 20)
			return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
		}
	}
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchTracks(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyTracks(page.Tracks)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchAlbums(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchAlbums(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtistAlbums(page.Albums)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchArtists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchArtists(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtistsPage(page.Artists)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Artists, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetPlaylistTracks(request common.MediaRequest) tea.Cmd {
	if utils.IsYouTubeMusicMode() || strings.HasPrefix(request.EntityURI, "localpl:") {
		return func() tea.Msg {
			plID := strings.TrimPrefix(request.EntityURI, "localpl:")
			pl, err := playlist.GetPlaylist(plID)
			if err != nil {
				return mediaLoadErrMsg{err: err, request: request}
			}
			var entities []common.Entity
			for _, t := range pl.Tracks {
				desc := t.Artist
				if t.Album != "" {
					desc += " • " + t.Album
				}
				id := fmt.Sprintf("ytmusic:%s|%s|%s|%s", t.VideoID, t.Title, t.Artist, t.ArtURL)
				entities = append(entities, common.NewEntity(t.Title, desc, id, t.ArtURL))
			}
			if len(entities) == 0 {
				entities = append(entities, common.NewEntity("Empty Playlist", "Play any song from search to build your library", "ytmusic:empty", ""))
			}
			pagination := paginationFromOffset(0, len(entities), len(entities), 50)
			return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
		}
	}
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		tracks, total, err := m.spotifyClient.GetPlaylistTracks(context.Background(), request.EntityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyTracks(tracks)
		pagination := paginationFromOffset(offset, len(entities), total, pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetArtistAlbums(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		albums, err := m.spotifyClient.GetArtistAlbums(context.Background(), request.EntityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtistAlbums(albums.Albums)
		pagination := paginationFromOffset(offset, len(entities), int(albums.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetAlbumTracks(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 50
		offset := decodeOffsetCursor(request.Cursor)
		tracks, err := m.spotifyClient.GetAlbumTracks(context.Background(), request.EntityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyAlbumTracks(tracks.Tracks)
		pagination := paginationFromOffset(offset, len(entities), int(tracks.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handlePlayTrackRequest(request common.MediaRequest) tea.Cmd {
	// 1. Play local playlist
	if strings.HasPrefix(request.EntityURI, "localpl:") {
		plID := strings.TrimPrefix(request.EntityURI, "localpl:")
		pl, err := playlist.GetPlaylist(plID)
		if err != nil || len(pl.Tracks) == 0 {
			return m.mediaCenter.SetStatus(request.PanelKind, "Playlist is empty")
		}
		m.ytQueue = pl.Tracks
		m.ytQueueIndex = 0
		m.updateQueueDisplay()
		m.mediaCenter.CloseLibrary()
		return m.playYtTrackCmd(pl.Tracks[0])
	}

	// 2. Play YouTube track
	if utils.IsYouTubeMusicMode() || strings.HasPrefix(request.EntityURI, "ytmusic:") {
		m.mediaCenter.SetDisplay("Streaming from YouTube Music...")
		m.mediaCenter.CloseLibrary()

		raw := strings.TrimPrefix(request.EntityURI, "ytmusic:")
		parts := strings.SplitN(raw, "|", 4)
		videoID := parts[0]
		title := "YouTube Track"
		artist := "YouTube Music"
		artURL := ""
		if len(parts) > 1 && parts[1] != "" {
			title = parts[1]
		}
		if len(parts) > 2 && parts[2] != "" {
			artist = parts[2]
		}
		if len(parts) > 3 && parts[3] != "" {
			artURL = parts[3]
		}

		currentTrack := ytmusic.Track{
			VideoID: videoID,
			Title:   title,
			Artist:  artist,
			ArtURL:  artURL,
		}

		// Auto-save to Favorites playlist for quick replay
		_ = playlist.AddTrackToPlaylist("favorites", currentTrack)

		m.lastLyricsTrack = title
		m.lastLyricsArtist = artist
		m.songInfo = common.SongInfo{
			Title:    title,
			Artist:   artist,
			Position: 0,
		}
		m.mediaCenter.SetDisplayFromSong(m.songInfo)
		m.playing = true
		m.playerReady = true
		m.updatePlayerStatus()

		playCmd := func() tea.Msg {
			streamURL := ytmusic.GetClient().GetStreamURL(videoID)
			if m.mpvPlayer != nil {
				if err := m.mpvPlayer.Play(streamURL); err != nil {
					return playTrackErrMsg{err: err, panelKind: request.PanelKind}
				}
			}
			return playTrackOkMsg{panelKind: request.PanelKind}
		}

		cmds := []tea.Cmd{
			playCmd,
			m.fetchLyricsCmd(title, artist),
			m.waitForMpvTrackEndCmd(),
		}

		// If user selected a track inside a playlist, play through playlist!
		if strings.HasPrefix(request.ContextURI, "localpl:") {
			plID := strings.TrimPrefix(request.ContextURI, "localpl:")
			if pl, err := playlist.GetPlaylist(plID); err == nil && len(pl.Tracks) > 0 {
				m.ytQueue = pl.Tracks
				m.ytQueueIndex = 0
				for idx, t := range pl.Tracks {
					if t.VideoID == videoID {
						m.ytQueueIndex = idx
						break
					}
				}
				m.updateQueueDisplay()
			} else {
				cmds = append(cmds, m.seedRadioCmd(videoID, currentTrack))
			}
		} else {
			cmds = append(cmds, m.seedRadioCmd(videoID, currentTrack))
		}

		if artURL != "" {
			artCols, artRows := m.desiredArtworkDimensions()
			m.lastArtworkURL = artURL
			cmds = append(cmds, m.fetchArtworkCmd(artURL, artCols, artRows))
		}

		return tea.Batch(cmds...)
	}

	if m.player == nil {
		return m.mediaCenter.SetStatus(request.PanelKind, "Player not ready")
	}
	m.mediaCenter.SetDisplay("Loading track...")
	m.mediaCenter.CloseLibrary()
	return func() tea.Msg {
		err := m.player.PlayTrack(context.Background(), request.EntityURI, request.ContextURI)
		if err != nil {
			return playTrackErrMsg{err: err, panelKind: request.PanelKind}
		}
		return playTrackOkMsg{panelKind: request.PanelKind}
	}
}

func adaptSpotifyPlaylists(page *spotify.SimplePlaylistPage) []common.Entity {
	logger.Log.Info().Any("p", page).Msg("adapt spotify playlist page")
	return common.MapSlice(page.Playlists, func(pl spotify.SimplePlaylist) common.Entity {
		return common.NewEntity(pl.Name, pl.Description, string(pl.URI), imageURL(pl.Images))
	})
}

func adaptSpotifySavedTracks(page *spotify.SavedTrackPage) []common.Entity {
	return common.MapSlice(page.Tracks, func(savedTrack spotify.SavedTrack) common.Entity {
		track := savedTrack.FullTrack
		desc := strings.TrimSpace(joinArtists(track.Artists))
		if track.Album.Name != "" {
			if desc != "" {
				desc += " • " + track.Album.Name
			} else {
				desc = track.Album.Name
			}
		}
		return common.NewEntity(track.Name, desc, string(track.URI), imageURL(track.Album.Images))
	})
}

func adaptSpotifySavedAlbums(page *spotify.SavedAlbumPage) []common.Entity {
	return common.MapSlice(page.Albums, func(savedAlbum spotify.SavedAlbum) common.Entity {
		album := savedAlbum.FullAlbum
		return common.NewEntity(album.Name, joinArtists(album.Artists), string(album.URI), imageURL(album.Images))
	})
}

func adaptSpotifyArtists(page *spotify.FullArtistCursorPage) []common.Entity {
	return adaptSpotifyArtistsPage(page.Artists)
}

func adaptSpotifyArtistsPage(artists []spotify.FullArtist) []common.Entity {
	return common.MapSlice(artists, func(artist spotify.FullArtist) common.Entity {
		desc := ""
		if len(artist.Genres) > 0 {
			desc = strings.Join(artist.Genres, ", ")
		}
		return common.NewEntity(artist.Name, desc, string(artist.URI), imageURL(artist.Images))
	})
}

func adaptSpotifyTracks(tracks []spotify.FullTrack) []common.Entity {
	return common.MapSlice(tracks, func(track spotify.FullTrack) common.Entity {
		desc := strings.TrimSpace(joinArtists(track.Artists))
		if track.Album.Name != "" {
			if desc != "" {
				desc += " • " + track.Album.Name
			} else {
				desc = track.Album.Name
			}
		}
		return common.NewEntity(track.Name, desc, string(track.URI), imageURL(track.Album.Images))
	})
}

func adaptSpotifyArtistAlbums(albums []spotify.SimpleAlbum) []common.Entity {
	return common.MapSlice(albums, func(album spotify.SimpleAlbum) common.Entity {
		return common.NewEntity(album.Name, joinArtists(album.Artists), string(album.URI), imageURL(album.Images))
	})
}

func adaptSpotifyAlbumTracks(tracks []spotify.SimpleTrack) []common.Entity {
	return common.MapSlice(tracks, func(track spotify.SimpleTrack) common.Entity {
		return common.NewEntity(track.Name, strings.TrimSpace(joinArtists(track.Artists)), string(track.URI), "")
	})
}

func imageURL(images []spotify.Image) string {
	if len(images) == 0 {
		return ""
	}
	return images[0].URL
}

func joinArtists(artists []spotify.SimpleArtist) string {
	if len(artists) == 0 {
		return ""
	}
	names := common.MapSlice(artists, func(artist spotify.SimpleArtist) string {
		return artist.Name
	})
	return strings.Join(names, ", ")
}
