package mediacenter

import (
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
		if maxH == 0 || maxH >= 34 {
			m.nowPlaying.SetSize(totalW, 11)
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
