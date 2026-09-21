package lyrics

import (
	"strings"
	"unicode/utf8"

	"cassette/core/lyrics"
	"cassette/core/theme"
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
	if w < 24 {
		w = 24
	}
	h := m.height
	if h < 10 {
		h = 10
	}

	th := theme.Get()
	cGray := lipgloss.NewStyle().Foreground(th.BorderColor())
	cCyan := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	cActive := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	cActiveWord := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true).Reverse(true)
	cSungWord := lipgloss.NewStyle().Foreground(th.PrimaryColor())
	cFutureWord := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cFuture := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cPast := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
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
	maxTextW := innerW - 3
	if maxTextW < 10 {
		maxTextW = 10
	}

	type displayRow struct {
		prefix   string
		text     string
		style    lipgloss.Style
		isKaraoke bool
		words    []lyrics.LyricWord
	}

	var allRows []displayRow
	activeRowIdx := 0

	if m.loading {
		allRows = append(allRows, displayRow{prefix: "", text: "Searching lyrics...", style: cNone})
	} else if m.lyrics == nil || (len(m.lyrics.SyncedLyrics) == 0 && len(m.lyrics.PlainLyrics) == 0) {
		allRows = append(allRows, displayRow{prefix: "", text: "No lyrics available", style: cNone})
		if m.trackName != "" {
			for _, chunk := range wrapWords(m.trackName, innerW) {
				allRows = append(allRows, displayRow{prefix: "", text: chunk, style: cNone})
			}
		}
	} else if m.lyrics.HasSynced {
		synced := m.lyrics.SyncedLyrics
		// Find active line
		activeLineIdx := 0
		for i, line := range synced {
			if line.TimeMs <= m.positionMs {
				activeLineIdx = i
			} else {
				break
			}
		}

		for i, line := range synced {
			trimmed := strings.TrimSpace(line.Text)
			if trimmed == "" {
				allRows = append(allRows, displayRow{prefix: "", text: "", style: cPast})
				continue
			}

			isActive := (i == activeLineIdx)
			isPast := (i < activeLineIdx)

			style := cFuture
			if isActive {
				style = cActive
				activeRowIdx = len(allRows)
			} else if isPast {
				style = cPast
			}

			// Check if this line has word-level timing for karaoke
			hasWords := isActive && len(line.Words) > 0 && m.lyrics.HasWordSync

			chunks := wrapWords(trimmed, maxTextW)
			for chunkIdx, chunk := range chunks {
				prefix := "  "
				if isActive {
					if chunkIdx == 0 {
						prefix = "▶ "
					} else {
						prefix = "   "
					}
				} else if chunkIdx > 0 {
					prefix = "   "
				}
				row := displayRow{
					prefix: prefix,
					text:   chunk,
					style:  style,
				}
				// Only render karaoke on the first chunk of the active line
				if hasWords && chunkIdx == 0 {
					row.isKaraoke = true
					row.words = line.Words
				}
				allRows = append(allRows, row)
			}
		}
	} else {
		// Plain lyrics
		for _, line := range m.lyrics.PlainLyrics {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				allRows = append(allRows, displayRow{prefix: "", text: "", style: cPast})
				continue
			}
			chunks := wrapWords(trimmed, innerW)
			for _, chunk := range chunks {
				allRows = append(allRows, displayRow{
					prefix: "",
					text:   chunk,
					style:  cFuture,
				})
			}
		}
	}

	// Calculate visible slice centered on active line
	start := 0
	if m.lyrics != nil && m.lyrics.HasSynced {
		centerRow := innerH / 2
		start = activeRowIdx - centerRow + m.scrollOff
		if start < 0 {
			start = 0
		}
		if len(allRows) > innerH && start > len(allRows)-innerH {
			start = len(allRows) - innerH
		}
	} else {
		start = m.scrollOff
		if start < 0 {
			start = 0
		}
	}

	var contentLines []string
	for r := 0; r < innerH; r++ {
		idx := start + r
		if idx < len(allRows) {
			row := allRows[idx]
			var rendered string
			if row.isKaraoke && len(row.words) > 0 {
				// Word-by-word karaoke rendering
				rendered = row.prefix + m.renderKaraokeWords(row.words, cSungWord, cActiveWord, cFutureWord)
			} else {
				rendered = row.style.Render(row.prefix + row.text)
			}
			contentLines = append(contentLines, rendered)
		} else {
			contentLines = append(contentLines, "")
		}
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

// renderKaraokeWords renders individual words with karaoke highlighting based on current position.
func (m *Model) renderKaraokeWords(words []lyrics.LyricWord, cSung, cActive, cFuture lipgloss.Style) string {
	var parts []string
	for _, w := range words {
		text := w.Text
		if m.positionMs >= w.EndMs {
			// Already sung
			parts = append(parts, cSung.Render(text))
		} else if m.positionMs >= w.StartMs {
			// Currently being sung
			parts = append(parts, cActive.Render(text))
		} else {
			// Upcoming
			parts = append(parts, cFuture.Render(text))
		}
	}
	return strings.Join(parts, " ")
}

func wrapWords(text string, maxW int) []string {
	if maxW <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var res []string
	curr := words[0]

	for _, w := range words[1:] {
		if utf8.RuneCountInString(curr)+1+utf8.RuneCountInString(w) <= maxW {
			curr += " " + w
		} else {
			res = append(res, curr)
			curr = w
		}
	}
	res = append(res, curr)
	return res
}
