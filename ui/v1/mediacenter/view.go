package mediacenter

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m *Model) View(maxW, maxH int) string {
	playerView := m.player.View(m.zenMode)
	playerW, playerH := lipgloss.Size(playerView)

	// If narrow viewport and library is open, collapse player to library
	if m.leftPanel == LeftPanelLibrary && maxW > 0 && maxW < playerW {
		m.mediaPanel.SetSize(maxW, maxH)
		return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(m.mediaPanel.View())
	}

	lyricsW := 30
	queueW := 30
	showLeft := m.leftPanel != LeftPanelClosed
	showRight := m.queueOpen

	// Adjust layout based on available viewport width
	if maxW > 0 {
		if maxW < playerW {
			showLeft = false
			showRight = false
		} else if maxW < playerW+30 {
			showLeft = false
			showRight = false
		} else if maxW < playerW+60 {
			if showLeft && showRight {
				showRight = false
			}
			if showLeft {
				lyricsW = max(30, min(58, maxW-playerW-2))
			} else if showRight {
				queueW = max(30, min(40, maxW-playerW-2))
			}
		} else {
			avail := maxW - playerW
			queueW = min(36, max(30, avail/3))
			calcLyrics := avail - queueW - 2
			if calcLyrics > 58 {
				calcLyrics = 58
			}
			if calcLyrics > lyricsW {
				lyricsW = calcLyrics
			}
		}
	}

	var leftView string
	if showLeft {
		if m.leftPanel == LeftPanelLyrics {
			m.lyricsModel.SetSize(lyricsW, playerH)
			leftView = m.lyricsModel.View()
		} else if m.leftPanel == LeftPanelLibrary {
			m.mediaPanel.SetSize(lyricsW, playerH)
			leftView = m.mediaPanel.View()
		} else if m.leftPanel == LeftPanelArtwork {
			leftView = m.renderArtworkPanel(lyricsW, playerH)
		}
	}

	var rightView string
	if showRight {
		m.queueModel.SetSize(queueW, playerH)
		rightView = m.queueModel.View()
	}

	var panels []string
	if leftView != "" {
		panels = append(panels, leftView)
	}
	panels = append(panels, playerView)
	if rightView != "" {
		panels = append(panels, rightView)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, panels...)
	totalW := lipgloss.Width(content)

	if !m.zenMode {
		m.displayScreen.SetSize(totalW, 3)
		content = lipgloss.JoinVertical(lipgloss.Left, m.displayScreen.View(), content)

		// Render Now Playing bottom bar with album art if vertical space permits
		if maxH == 0 || maxH >= 32 {
			npH := 16
			if maxH > 0 {
				contentH := lipgloss.Height(content)
				avail := maxH - contentH - 3 // leave room for borders and footer keybindings
				if avail < 10 {
					npH = 10
				} else if avail > 22 {
					npH = 22
				} else {
					npH = avail
				}
			}
			m.nowPlaying.SetSize(totalW, npH)
			npView := m.nowPlaying.View()
			if npView != "" {
				content = lipgloss.JoinVertical(lipgloss.Left, content, npView)
			}
		}
	}

	v := lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(content)
	w, h := lipgloss.Size(v)
	if ((w > maxW && maxW > 0) || (h > maxH && maxH > 0)) && m.leftPanel == LeftPanelLibrary {
		m.mediaPanel.SetSize(maxW, maxH)
		return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(m.mediaPanel.View())
	}
	return v
}

func (m *Model) renderArtworkPanel(w, h int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Width(w).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center)

	header := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render(" ALBUM ART ")

	var artBlock string
	if strings.TrimSpace(m.artANSI) != "" {
		lines := strings.Split(m.artANSI, "\n")
		maxLines := h - 4
		if maxLines < 6 {
			maxLines = 6
		}
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
		artBlock = strings.Join(lines, "\n")
	} else {
		artBlock = lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render("(no album art loaded)")
	}

	track := m.currentSong.Title
	if track == "" {
		track = "Not Playing"
	}
	artist := m.currentSong.Artist

	cTrack := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	cArtist := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	meta := cTrack.Render(truncateStr(track, w-4))
	if artist != "" {
		meta = meta + "\n" + cArtist.Render(truncateStr(artist, w-4))
	}

	content := lipgloss.JoinVertical(lipgloss.Center, header, "", artBlock, "", meta)
	return boxStyle.Render(content)
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
