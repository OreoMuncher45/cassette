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

	sideW := 30
	showLeft := m.leftPanel != LeftPanelClosed
	showRight := m.queueOpen

	// Adjust layout based on available viewport width
	if maxW > 0 {
		if maxW < playerW {
			showLeft = false
			showRight = false
		} else if maxW < playerW+sideW {
			showLeft = false
			showRight = false
		} else if maxW < playerW+2*sideW {
			if showLeft && showRight {
				showRight = false
			}
		} else if maxW >= playerW+68 {
			calcSide := (maxW - playerW) / 2
			if calcSide > 36 {
				calcSide = 36
			}
			if calcSide > sideW {
				sideW = calcSide
			}
		}
	}

	var leftView string
	if showLeft {
		if m.leftPanel == LeftPanelLyrics {
			m.lyricsModel.SetSize(sideW, playerH)
			leftView = m.lyricsModel.View()
		} else if m.leftPanel == LeftPanelLibrary {
			m.mediaPanel.SetSize(sideW, playerH)
			leftView = m.mediaPanel.View()
		}
	}

	var rightView string
	if showRight {
		m.queueModel.SetSize(sideW, playerH)
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
	}

	v := lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(content)
	w, h := lipgloss.Size(v)
	if ((w > maxW && maxW > 0) || (h > maxH && maxH > 0)) && m.leftPanel == LeftPanelLibrary {
		m.mediaPanel.SetSize(maxW, maxH)
		return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(m.mediaPanel.View())
	}
	return v
}
