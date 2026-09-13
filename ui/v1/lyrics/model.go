package lyrics

import (
	"strings"
	"unicode/utf8"

	"cassette/core/lyrics"
	"charm.land/lipgloss/v2"
)

type Model struct {
	width      int
	height     int
	lyrics     *lyrics.Lyrics
	positionMs int
	loading    bool
	trackName  string
	artistName string
	scrollOff  int
}

func NewModel() Model {
	return Model{
		width:  32,
		height: 22,
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetTrack(track, artist string) {
	if m.trackName != track || m.artistName != artist {
		m.trackName = track
		m.artistName = artist
		m.lyrics = nil
		m.loading = true
		m.scrollOff = 0
	}
}

func (m *Model) SetLyrics(l *lyrics.Lyrics) {
	m.lyrics = l
	m.loading = false
	m.scrollOff = 0
}

func (m *Model) SetPosition(posMs int) {
	m.positionMs = posMs
}

func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

func (m *Model) ScrollUp() {
	if m.scrollOff > 0 {
		m.scrollOff--
	}
}

func (m *Model) ScrollDown() {
	m.scrollOff++
}

func (m *Model) View() string {
	w := m.width
	if w < 20 {
		w = 20
	}
	h := m.height
	if h < 10 {
		h = 10
	}

	cGray := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	cCyan := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	cActive := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	cDim := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cNone := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Header: ╭── LYRICS ────────────────────╮
	title := " LYRICS "
	topRuleW := w - 2 - 3 - lipgloss.Width(title)
	if topRuleW < 2 {
		topRuleW = 2
	}
	header := cGray.Render("╭──") + cCyan.Render(title) + cGray.Render(strings.Repeat("─", topRuleW)+"╮")
	footer := cGray.Render("╰" + strings.Repeat("─", w-2) + "╯")

	innerW := w - 4
	innerH := h - 2

	var contentLines []string

	if m.loading {
		contentLines = append(contentLines, cNone.Render("Searching lyrics..."))
	} else if m.lyrics == nil || (len(m.lyrics.SyncedLyrics) == 0 && len(m.lyrics.PlainLyrics) == 0) {
		contentLines = append(contentLines, cNone.Render("No lyrics available"))
		if m.trackName != "" {
			contentLines = append(contentLines, cNone.Render(truncate(m.trackName, innerW)))
		}
	} else if m.lyrics.HasSynced {
		synced := m.lyrics.SyncedLyrics
		// Find active line
		activeIdx := 0
		for i, line := range synced {
			if line.TimeMs <= m.positionMs {
				activeIdx = i
			} else {
				break
			}
		}

		centerRow := innerH / 2
		start := activeIdx - centerRow + m.scrollOff
		if start < 0 {
			start = 0
		}

		for r := 0; r < innerH; r++ {
			idx := start + r
			if idx >= len(synced) {
				contentLines = append(contentLines, "")
				continue
			}

			text := truncate(synced[idx].Text, innerW-3)
			if idx == activeIdx {
				contentLines = append(contentLines, cActive.Render("▶ "+text))
			} else {
				contentLines = append(contentLines, cDim.Render("  "+text))
			}
		}
	} else {
		// Plain lyrics
		plain := m.lyrics.PlainLyrics
		start := m.scrollOff
		if start < 0 {
			start = 0
		}
		for r := 0; r < innerH; r++ {
			idx := start + r
			if idx >= len(plain) {
				contentLines = append(contentLines, "")
			} else {
				contentLines = append(contentLines, cDim.Render(truncate(plain[idx], innerW)))
			}
		}
	}

	// Pad/crop to innerH
	for len(contentLines) < innerH {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > innerH {
		contentLines = contentLines[:innerH]
	}

	var renderedRows []string
	renderedRows = append(renderedRows, header)
	for _, l := range contentLines {
		lw := lipgloss.Width(l)
		rightSpaces := innerW - lw
		if rightSpaces < 0 {
			rightSpaces = 0
		}
		row := cGray.Render("│ ") + l + strings.Repeat(" ", rightSpaces) + cGray.Render(" │")
		renderedRows = append(renderedRows, row)
	}
	renderedRows = append(renderedRows, footer)

	return strings.Join(renderedRows, "\n")
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
