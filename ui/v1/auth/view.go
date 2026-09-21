package auth

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Authentication failed: %v\nPress [s/Esc] to switch to YouTube Music, or [q] to exit...", m.err))
	}

	w := m.width
	if w <= 0 {
		w = 74
	}

	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Underline(true)
	boxBorder := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	boxW := 72
	if w < boxW+4 {
		boxW = max(40, w-4)
	}

	innerW := boxW - 4

	var lines []string
	lines = append(lines, boxBorder.Render("╭"+strings.Repeat("─", boxW-2)+"╮"))
	lines = append(lines, boxBorder.Render("│ ")+accent.Render(centerStr("CASSETTE • AUTHENTICATION", innerW))+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+strings.Repeat(" ", innerW)+boxBorder.Render(" │"))

	if m.auth.AuthServer.Started.Load() {
		lines = append(lines, boxBorder.Render("│ ")+bright.Render(padStr("Spotify Login Required (Spotify Mode)", innerW))+boxBorder.Render(" │"))
		lines = append(lines, boxBorder.Render("│ ")+dim.Render(padStr("Please open this URL in your browser to authorize:", innerW))+boxBorder.Render(" │"))
		lines = append(lines, boxBorder.Render("│ ")+strings.Repeat(" ", innerW)+boxBorder.Render(" │"))

		url := m.auth.GetAuthURL()
		if len(url) > innerW {
			lines = append(lines, boxBorder.Render("│ ")+urlStyle.Render(url[:innerW])+boxBorder.Render(" │"))
			if len(url) > innerW*2 {
				lines = append(lines, boxBorder.Render("│ ")+urlStyle.Render(url[innerW:innerW*2])+boxBorder.Render(" │"))
			} else {
				lines = append(lines, boxBorder.Render("│ ")+urlStyle.Render(padStr(url[innerW:], innerW))+boxBorder.Render(" │"))
			}
		} else {
			lines = append(lines, boxBorder.Render("│ ")+urlStyle.Render(padStr(url, innerW))+boxBorder.Render(" │"))
		}

		lines = append(lines, boxBorder.Render("│ ")+strings.Repeat(" ", innerW)+boxBorder.Render(" │"))

		copyHint := "[c] Copy URL to clipboard"
		if m.copied {
			copyHint = "✓ Copied to clipboard!"
			lines = append(lines, boxBorder.Render("│ ")+green.Render(padStr(copyHint, innerW))+boxBorder.Render(" │"))
		} else {
			lines = append(lines, boxBorder.Render("│ ")+dim.Render(padStr(copyHint, innerW))+boxBorder.Render(" │"))
		}
	} else {
		lines = append(lines, boxBorder.Render("│ ")+bright.Render(padStr("Starting Spotify authorization listener...", innerW))+boxBorder.Render(" │"))
		lines = append(lines, boxBorder.Render("│ ")+dim.Render(padStr("Waiting for local callback on 127.0.0.1:8287", innerW))+boxBorder.Render(" │"))
	}

	lines = append(lines, boxBorder.Render("│ ")+strings.Repeat(" ", innerW)+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+boxBorder.Render(strings.Repeat("─", innerW))+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+strings.Repeat(" ", innerW)+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+green.Render(padStr("  ▶ Press [s / y / Esc] to SKIP Spotify and use YouTube Music", innerW))+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+dim.Render(padStr("     (Free playback, zero login or API keys required)", innerW))+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+strings.Repeat(" ", innerW)+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("│ ")+dim.Render(padStr("  ▶ Press [q] or [Ctrl+C] to quit", innerW))+boxBorder.Render(" │"))
	lines = append(lines, boxBorder.Render("╰"+strings.Repeat("─", boxW-2)+"╯"))

	box := strings.Join(lines, "\n")
	view := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(box)
	return tea.NewView(view)
}

func centerStr(s string, w int) string {
	runeLen := len([]rune(s))
	if runeLen >= w {
		return s
	}
	left := (w - runeLen) / 2
	right := w - runeLen - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func padStr(s string, w int) string {
	runeLen := len([]rune(s))
	if runeLen >= w {
		return s
	}
	return s + strings.Repeat(" ", w-runeLen)
}
