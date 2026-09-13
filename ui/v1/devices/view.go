package devices

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) View() tea.View {
	const boxW = 60

	frameBorder := lipgloss.NewStyle().Foreground(lipgloss.BrightCyan)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightGreen)
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	itemNormal := lipgloss.NewStyle().Foreground(lipgloss.BrightWhite)
	itemSelected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightYellow)
	activeBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightGreen)
	typeBadge := lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightRed)

	var innerLines []string

	// Header
	innerLines = append(innerLines, titleStyle.Width(boxW-4).Align(lipgloss.Center).Render("SPOTIFY CONNECT"))
	innerLines = append(innerLines, subtitleStyle.Width(boxW-4).Align(lipgloss.Center).Render("SELECT PLAYBACK DEVICE"))
	innerLines = append(innerLines, frameBorder.Render(strings.Repeat("─", boxW-4)))
	innerLines = append(innerLines, "")

	switch m.state {
	case StateLoading:
		innerLines = append(innerLines, "")
		innerLines = append(innerLines, itemNormal.Width(boxW-4).Align(lipgloss.Center).Render("Scanning for Spotify Connect devices..."))
		innerLines = append(innerLines, "")
		innerLines = append(innerLines, hintStyle.Width(boxW-4).Align(lipgloss.Center).Render("Make sure Spotify, spotifyd, or Web Player is running"))
		innerLines = append(innerLines, "")

	case StateNoDevices:
		innerLines = append(innerLines, warnStyle.Width(boxW-4).Align(lipgloss.Center).Render("NO ACTIVE DEVICE FOUND"))
		innerLines = append(innerLines, "")
		innerLines = append(innerLines, itemNormal.Width(boxW-4).Align(lipgloss.Center).Render("Spotify Connect requires an active session to control."))
		innerLines = append(innerLines, subtitleStyle.Width(boxW-4).Align(lipgloss.Center).Render("Launch or play Spotify on any device:"))
		innerLines = append(innerLines, hintStyle.Width(boxW-4).Align(lipgloss.Center).Render("• spotifyd (local daemon / audio engine)"))
		innerLines = append(innerLines, hintStyle.Width(boxW-4).Align(lipgloss.Center).Render("• Spotify Web Player (open.spotify.com)"))
		innerLines = append(innerLines, hintStyle.Width(boxW-4).Align(lipgloss.Center).Render("• Mobile / Desktop Spotify Client"))
		if m.statusMessage != "" {
			innerLines = append(innerLines, "")
			innerLines = append(innerLines, warnStyle.Width(boxW-4).Align(lipgloss.Center).Render(m.statusMessage))
		}
		innerLines = append(innerLines, "")

	case StateSelecting:
		for i, dev := range m.devices {
			prefix := "  "
			cursorMark := " "
			lineStyle := itemNormal
			if i == m.cursor {
				prefix = "> "
				cursorMark = "►"
				lineStyle = itemSelected
			}

			activeMark := "○"
			statusText := ""
			if dev.Active {
				activeMark = "●"
				statusText = activeBadge.Render("ACTIVE")
			}
			if dev.Restricted {
				statusText = warnStyle.Render("RESTRICTED")
			}

			typeStr := fmt.Sprintf("[%s]", strings.ToUpper(dev.Type))
			if len(typeStr) > 13 {
				typeStr = typeStr[:13]
			}
			paddedType := fmt.Sprintf("%-13s", typeStr)

			name := dev.Name
			if len(name) > 22 {
				name = name[:19] + "..."
			}
			paddedName := fmt.Sprintf("%-22s", name)

			renderedLine := fmt.Sprintf("%s%s %s %s %s",
				lineStyle.Render(prefix+cursorMark),
				activeBadge.Render(activeMark),
				typeBadge.Render(paddedType),
				lineStyle.Render(paddedName),
				statusText,
			)
			innerLines = append(innerLines, renderedLine)
		}
		innerLines = append(innerLines, "")
	}

	// Footer / Hotkeys
	innerLines = append(innerLines, frameBorder.Render(strings.Repeat("─", boxW-4)))
	var helpText string
	if m.state == StateNoDevices {
		helpText = "[r] Rescan Devices   [q] Quit"
		if m.canDismiss {
			helpText += "   [Esc/d] Dismiss"
		}
	} else {
		helpText = "[↑/↓/j/k] Navigate   [Enter] Select   [r] Rescan   [q] Quit"
		if m.canDismiss {
			helpText += "   [Esc/d] Dismiss"
		}
	}
	innerLines = append(innerLines, hintStyle.Width(boxW-4).Align(lipgloss.Center).Render(helpText))

	content := strings.Join(innerLines, "\n")
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.BrightCyan).
		Padding(1, 2).
		Width(boxW).
		Render(content)

	view := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)

	return tea.NewView(view)
}
