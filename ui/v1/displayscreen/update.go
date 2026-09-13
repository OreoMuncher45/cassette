package displayscreen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	ansi "github.com/charmbracelet/x/ansi"
	"cassette/core/ticker"
	"cassette/ui/v1/common"
)

const scrollGap = "   "

func (m *Model) SetDisplayFromSong(song common.SongInfo) {
	if song.Title == "" {
		return
	}
	separator := " • "
	m.display = m.styles.primary.Render(song.Title) +
		m.styles.accent.Render(separator) +
		m.styles.muted.Render(song.Artist) +
		m.styles.accent.Render(separator) +
		m.styles.muted.Render(song.Album)
	m.scrollOffset = 0
}

func (m *Model) Update(tea.Msg) tea.Cmd {
	return nil
}

func (m *Model) NextFrame() tea.Cmd {
	if span := m.scrollSpan(m.display); span > 0 {
		m.scrollOffset = (m.scrollOffset + 1) % span
	}
	return ticker.DoTick()
}

func (m *Model) scrollText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(text) <= width {
		return text
	}

	base := text + scrollGap
	track := base + base
	span := m.scrollSpan(text)
	if span == 0 {
		return strings.Repeat(" ", width)
	}

	start := m.scrollOffset % span
	return ansi.Cut(track, start, start+width)
}

func (m *Model) scrollSpan(text string) int {
	if text == "" {
		return 0
	}
	return ansi.StringWidth(text + scrollGap)
}
