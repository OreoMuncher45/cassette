package queue

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"cassette/core/theme"
	"charm.land/lipgloss/v2"
	"github.com/zmb3/spotify/v2"
)

type Model struct {
	width     int
	height    int
	queue     *spotify.Queue
	scrollOff int
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

func (m *Model) SetQueue(q *spotify.Queue) {
	m.queue = q
}

func (m *Model) ScrollUp() {
	if m.scrollOff > 0 {
		m.scrollOff--
	}
}

func (m *Model) ScrollDown() {
	if m.queue != nil && m.scrollOff < len(m.queue.Items)-1 {
		m.scrollOff++
	}
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

	th := theme.Get()
	cGray := lipgloss.NewStyle().Foreground(th.BorderColor())
	cCyan := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	cTrack := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	cArtist := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cNone := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Header: ╭── QUEUE ─────────────────────╮
	title := " QUEUE "
	topRuleW := w - 2 - 3 - lipgloss.Width(title)
	if topRuleW < 2 {
		topRuleW = 2
	}
	header := cGray.Render("╭──") + cCyan.Render(title) + cGray.Render(strings.Repeat("─", topRuleW)+"╮")
	footer := cGray.Render("╰" + strings.Repeat("─", w-2) + "╯")

	innerW := w - 4
	innerH := h - 2

	var contentLines []string

	if m.queue == nil || len(m.queue.Items) == 0 {
		contentLines = append(contentLines, cNone.Render("Queue is empty"))
		contentLines = append(contentLines, "")
		contentLines = append(contentLines, cArtist.Render("Tracks added to queue"))
		contentLines = append(contentLines, cArtist.Render("or endless song radio"))
		contentLines = append(contentLines, cArtist.Render("will appear here."))
	} else {
		items := m.queue.Items
		start := m.scrollOff
		if start < 0 {
			start = 0
		}
		if start >= len(items) {
			start = 0
		}

		itemIdx := start
		for len(contentLines) < innerH && itemIdx < len(items) {
			item := items[itemIdx]
			dur := formatDuration(int(item.Duration))

			artist := ""
			if len(item.Artists) > 0 {
				artist = item.Artists[0].Name
			}

			// Line 1: 1. Track Name
			numStr := fmt.Sprintf("%d. ", itemIdx+1)
			trackMax := innerW - len(numStr)
			tLine := cCyan.Render(numStr) + cTrack.Render(truncate(item.Name, trackMax))
			contentLines = append(contentLines, tLine)

			// Line 2:    Artist • 03:45
			metaStr := fmt.Sprintf("   %s • %s", truncate(artist, innerW-11), dur)
			contentLines = append(contentLines, cArtist.Render(metaStr))

			itemIdx++
		}
	}

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

func formatDuration(ms int) string {
	if ms <= 0 {
		return "00:00"
	}
	totalSec := ms / 1000
	m := totalSec / 60
	s := totalSec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
