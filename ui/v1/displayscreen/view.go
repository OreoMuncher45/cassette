package displayscreen

import (
	"cassette/core/theme"
	"charm.land/lipgloss/v2"
)

func (m *Model) View() string {
	raw := m.display
	contentWidth := max(0, m.width-2)
	th := theme.Get()
	cPrim := th.PrimaryColor()
	cBorder := th.BorderColor()

	styled := lipgloss.NewStyle().Foreground(cPrim).Bold(true).Render(raw)
	if contentWidth > 0 {
		if lipgloss.Width(raw) > contentWidth {
			styled = lipgloss.NewStyle().Foreground(cPrim).Bold(true).Render(m.scrollText(raw, contentWidth))
		}
		styled = lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(styled)
	}
	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(cBorder).
		Width(m.width).
		Height(m.height)
	return panelStyle.Render(styled)
}

