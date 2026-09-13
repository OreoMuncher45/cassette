package nowplaying

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"cassette/core/theme"
	"cassette/ui/v1/common"
	"charm.land/lipgloss/v2"
)

type Model struct {
	width      int
	height     int
	artANSI    string
	songInfo   common.SongInfo
	volumeInfo common.VolumeInfo
	playing    bool
	deviceName string
	shuffled   bool
}

func NewModel() Model {
	return Model{
		width:  100,
		height: 11,
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetArtwork(ansi string) {
	m.artANSI = ansi
}

func (m *Model) SetSong(song common.SongInfo) {
	m.songInfo = song
}

func (m *Model) SetVolume(v common.VolumeInfo) {
	m.volumeInfo = v
}

func (m *Model) SetStatus(playing bool, device string, shuffled bool) {
	m.playing = playing
	m.deviceName = device
	m.shuffled = shuffled
}

func (m *Model) View() string {
	w := m.width
	if w < 50 {
		return ""
	}

	th := theme.Get()
	cBorder := lipgloss.NewStyle().Foreground(th.BorderColor())
	cTrack := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	cArtist := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	cAlbum := lipgloss.NewStyle().Foreground(th.SecondaryColor())
	cMeta := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cStatus := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	if !m.playing {
		cStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Bold(true)
	}

	// Album Art Box (Square aspect ratio: width = height * 2)
	artH := m.height - 2
	if artH > 22 {
		artH = 22
	}
	if artH < 8 {
		artH = 8
	}
	artW := artH * 2

	var artBlock string
	if strings.TrimSpace(m.artANSI) != "" {
		lines := strings.Split(m.artANSI, "\n")
		if len(lines) > artH {
			lines = lines[:artH]
		} else if len(lines) < artH {
			diff := artH - len(lines)
			padTop := diff / 2
			padBot := diff - padTop
			var padded []string
			for i := 0; i < padTop; i++ {
				padded = append(padded, "")
			}
			padded = append(padded, lines...)
			for i := 0; i < padBot; i++ {
				padded = append(padded, "")
			}
			lines = padded
		}
		artBlock = strings.Join(lines, "\n")
	} else {
		// Retro placeholder
		artLines := []string{
			"  ╭──────────────╮  ",
			"  │   CASSETTE   │  ",
			"  │   HI-FI DECK │  ",
			"  │      (◉)     │  ",
			"  │    SPOTIFY   │  ",
			"  ╰──────────────╯  ",
		}
		for len(artLines) < artH {
			artLines = append(artLines, strings.Repeat(" ", artW))
		}
		artBlock = cBorder.Render(strings.Join(artLines, "\n"))
	}

	artBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.BorderColor()).
		Padding(0, 1)

	renderedArtBox := artBoxStyle.Render(artBlock)
	artBoxW := lipgloss.Width(renderedArtBox)

	// Right Details Box
	detailsW := w - artBoxW - 1
	if detailsW < 25 {
		return ""
	}

	innerDetailsW := detailsW - 4
	if innerDetailsW < 10 {
		innerDetailsW = 10
	}

	title := m.songInfo.Title
	if title == "" {
		title = "Not Playing"
	}
	title = truncate(title, innerDetailsW)

	artist := m.songInfo.Artist
	album := m.songInfo.Album
	metaLine := ""
	if artist != "" && album != "" {
		metaLine = cArtist.Render(truncate(artist, innerDetailsW/2)) + cMeta.Render(" • ") + cAlbum.Render(truncate(album, innerDetailsW/2))
	} else if artist != "" {
		metaLine = cArtist.Render(truncate(artist, innerDetailsW))
	}

	// Progress slider: 01:17 ━━━━━━━━━●━━━━━━━━━━━━━━ 03:12
	pos := m.songInfo.Position
	dur := m.songInfo.Duration
	posStr := formatMs(pos)
	durStr := formatMs(dur)
	sliderW := innerDetailsW - len(posStr) - len(durStr) - 4
	if sliderW < 6 {
		sliderW = 6
	}

	slider := renderSlider(pos, dur, sliderW, th)
	progressLine := cMeta.Render(posStr+" ") + slider + cMeta.Render(" "+durStr)

	// Status Line
	statusText := "▶ PLAYING"
	if !m.playing {
		statusText = "⏸ PAUSED"
	}
	dev := m.deviceName
	if dev == "" {
		dev = "cassette (PC)"
	}
	shufStr := "OFF"
	if m.shuffled {
		shufStr = "ON 🔀"
	}
	statusLine := fmt.Sprintf("%s  •  Device: %s  •  Vol: %d%%  •  Shuffle: %s",
		cStatus.Render(statusText),
		cMeta.Render(dev),
		m.volumeInfo.Volume,
		cMeta.Render(shufStr),
	)

	detailsLines := []string{
		cTrack.Render(title),
		metaLine,
		"",
		progressLine,
		"",
		statusLine,
	}

	// Pad details to art box height
	artBoxH := lipgloss.Height(renderedArtBox)
	for len(detailsLines) < artBoxH-2 {
		detailsLines = append(detailsLines, "")
	}

	detailsContent := strings.Join(detailsLines, "\n")
	renderedDetailsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.BorderColor()).
		Width(detailsW).
		Height(artBoxH).
		Padding(0, 1).
		Render(detailsContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedArtBox, renderedDetailsBox)
}

func renderSlider(pos, dur, width int, th *theme.Manager) string {
	if width <= 0 {
		return ""
	}
	fraction := 0.0
	if dur > 0 {
		fraction = float64(pos) / float64(dur)
		if fraction > 1.0 {
			fraction = 1.0
		}
		if fraction < 0.0 {
			fraction = 0.0
		}
	}

	knobPos := int(fraction * float64(width-1))
	if knobPos >= width {
		knobPos = width - 1
	}

	cDone := lipgloss.NewStyle().Foreground(th.PrimaryColor())
	cKnob := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	cRest := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var sb strings.Builder
	for i := 0; i < width; i++ {
		if i < knobPos {
			sb.WriteString(cDone.Render("━"))
		} else if i == knobPos {
			sb.WriteString(cKnob.Render("●"))
		} else {
			sb.WriteString(cRest.Render("─"))
		}
	}
	return sb.String()
}

func formatMs(ms int) string {
	if ms <= 0 {
		return "00:00"
	}
	totalSec := ms / 1000
	m := totalSec / 60
	s := totalSec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
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
