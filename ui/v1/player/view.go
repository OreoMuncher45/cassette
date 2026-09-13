package player

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m *Model) View(zenMode bool) string {
	cassetteView := m.cassette.View()
	if zenMode {
		return m.style.Render(cassetteView)
	}

	var buttonsView string
	for i := range m.controls {
		if i == len(m.controls)/2 {
			buttonsView = lipgloss.JoinHorizontal(lipgloss.Left, buttonsView, "  ", m.controls[i].View())
			continue
		}
		buttonsView = lipgloss.JoinHorizontal(lipgloss.Left, buttonsView, " ", m.controls[i].View())
	}

	cassetteW := lipgloss.Width(cassetteView)
	buttonsW := lipgloss.Width(buttonsView)
	pad := max(0, (cassetteW-buttonsW)/2)

	btnLines := strings.Split(buttonsView, "\n")
	for idx := range btnLines {
		btnLines[idx] = strings.Repeat(" ", pad) + btnLines[idx]
	}
	alignedButtons := strings.Join(btnLines, "\n")

	content := lipgloss.JoinVertical(lipgloss.Left, cassetteView, "", alignedButtons)
	return m.style.Render(content)
}

func playerShell(innerW int, innerH int) string {
	lines := make([]string, 0, innerH)
	lines = append(lines, strings.Repeat(" ", innerW))
	for range innerH - 2 {
		lines = append(lines, strings.Repeat(" ", innerW))
	}
	lines = append(lines, strings.Repeat(" ", innerW))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

type borderChars struct {
	topL string
	topR string
	botL string
	botR string
	mid  string
	fill string
}

func buttonShellBase(width int, height int, chars borderChars) string {
	lines := make([]string, height)
	lines[0] = chars.topL + strings.Repeat(chars.fill, width-2) + chars.topR
	for i := 1; i < height-1; i++ {
		lines[i] = chars.mid + strings.Repeat(" ", width-2) + chars.mid
	}
	lines[height-1] = chars.botL + strings.Repeat(chars.fill, width-2) + chars.botR
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func buttonShell(width int, height int, pressed bool) string {
	if pressed {
		return buttonShellBase(width, height, borderChars{"╔", "╗", "╚", "╝", "║", "═"})
	}
	return buttonShellBase(width, height, borderChars{"╭", "╮", "╰", "╯", "│", "─"})
}

func (b *button) View() string {
	const (
		btnW = 6
		btnH = 3
	)

	shellColor := lipgloss.BrightBlack
	iconStyle := b.style
	if b.pressed {
		shellColor = lipgloss.BrightGreen
		iconStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightWhite).Bold(true)
	}
	shell := lipgloss.NewStyle().Foreground(shellColor).Render(buttonShell(btnW, btnH, b.pressed))
	iconW := lipgloss.Width(b.icon)
	x := 1 + (btnW-2-iconW)/2
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(shell).ID("shell"),
		lipgloss.NewLayer(iconStyle.Render(b.icon)).X(x).Y(1).ID("icon"),
	}
	return lipgloss.NewCompositor(layers...).Render()
}
