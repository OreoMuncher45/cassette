package player

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"cassette/core/theme"
	"cassette/core/utils"
)

const (
	cassetteWidth = 74
)

type cassette struct {
	currentFrame int
	playerStatus cassetteStatus
}

type cassetteStatus struct {
	Online     bool
	Playing    bool
	ShowVolume bool
	CurrentMs  int
	DurationMs int
	Volume     int
	VolumeMax  int
	Shuffled   bool
	TrackName  string
	ArtistName string
}

func newCassette() cassette {
	return cassette{}
}

func (c *cassette) NextFrame() {
	if c.playerStatus.Playing {
		c.currentFrame = (c.currentFrame + 1) % 4
	}
}

// 4 rotating gear frames for the spools (each line is exactly 9 visual characters)
var spoolFrames = [4][5]string{
	// Frame 0 (teeth at 12, 2, 4, 6, 8, 10)
	{
		"  ▄███▄  ",
		" █▀ █ ▀█ ",
		"█ ▄ █ ▄ █",
		" █▄ █ ▄█ ",
		"  ▀███▀  ",
	},
	// Frame 1 (teeth rotated 30 deg)
	{
		"  ▄███▄  ",
		" █ ▀█▀ █ ",
		"██  █  ██",
		" █ ▄█▄ █ ",
		"  ▀███▀  ",
	},
	// Frame 2 (teeth rotated 60 deg)
	{
		"  ▄███▄  ",
		" █▄ █ ▄█ ",
		"█ ▀ █ ▀ █",
		" █▀ █ ▀█ ",
		"  ▀███▀  ",
	},
	// Frame 3 (teeth rotated 90 deg)
	{
		"  ▄███▄  ",
		" █ ▄█▄ █ ",
		"██  █  ██",
		" █ ▀█▀ █ ",
		"  ▀███▀  ",
	},
}

func (c *cassette) View() string {
	const W = cassetteWidth
	const innerLabelW = W - 8 // 66

	track := strings.TrimSpace(c.playerStatus.TrackName)
	artist := strings.TrimSpace(c.playerStatus.ArtistName)

	if track == "" && !c.playerStatus.Online {
		if utils.IsYouTubeMusicMode() {
			track = "CASSETTE TAPE • YOUTUBE MUSIC"
			artist = "PRESS / TO SEARCH OR SPACE TO PLAY"
		} else {
			track = "CASSETTE TAPE • SPOTIFY REMOTE"
			artist = "WAITING FOR SPOTIFY PLAYBACK"
		}
	} else if track == "" {
		track = "CASSETTE TAPE • READY TO PLAY"
		artist = "PRESS SPACE OR SEARCH (/) TO START"
	}

	prog := 0.0
	if c.playerStatus.DurationMs > 0 {
		prog = float64(c.playerStatus.CurrentMs) / float64(c.playerStatus.DurationMs)
		if prog > 1.0 {
			prog = 1.0
		}
	}

	// Active spool rotation frame
	frameIdx := 0
	if c.playerStatus.Playing {
		frameIdx = c.currentFrame % 4
	}

	spoolL := spoolFrames[frameIdx]
	spoolR := spoolFrames[frameIdx]

	// Tape thickness inside center window (supply reel vs take-up reel)
	// Both leftTape and rightTape are exactly 4 visual characters
	var leftTape, rightTape string
	if prog < 0.33 {
		leftTape = "███ "
		rightTape = "  █ "
	} else if prog < 0.66 {
		leftTape = " ██ "
		rightTape = " ██ "
	} else {
		leftTape = " █  "
		rightTape = " ███"
	}

	// Dynamic styles from theme engine (with breathing / rainbow support)
	th := theme.Get()
	cGray := lipgloss.NewStyle().Foreground(lipgloss.Color(th.BorderHex()))
	cDark := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	cCyan := lipgloss.NewStyle().Foreground(lipgloss.Color(th.PrimaryHex())).Bold(true)
	cWhite := lipgloss.NewStyle().Foreground(lipgloss.Color("254"))
	cTrack := lipgloss.NewStyle().Foreground(lipgloss.Color(th.SecondaryHex())).Bold(true)
	cGreen := lipgloss.NewStyle().Foreground(lipgloss.Color(th.PrimaryHex())).Bold(true)
	cArtist := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	cStatus := lipgloss.NewStyle().Foreground(lipgloss.Color(th.PrimaryHex())).Bold(true)

	// Line 0: Top outer shell with textured shading (width 74)
	l0 := cGray.Render("╭" + strings.Repeat("░", W-2) + "╮")

	// Line 1: Top screws and upper cassette header (width 74)
	l1 := cGray.Render("│ ") + cCyan.Render("(o)") + cGray.Render("  ") + cCyan.Render("╭───╮") + strings.Repeat(" ", W-17) + cCyan.Render("(o)") + cGray.Render(" │")

	// Line 2: Side B indicator, top rule, and source badge (width 74)
	sourceBadge := "[YT MUSIC]  "
	if !utils.IsYouTubeMusicMode() {
		sourceBadge = "[SPOTIFY]   "
	}
	l2 := cGray.Render("│      ") + cCyan.Render("│ B │") + cGray.Render("  "+strings.Repeat("─", W-32)+"  ") + cCyan.Render(sourceBadge) + cGray.Render("   │")

	// Line 3: Side B box bottom (width 74)
	l3 := cGray.Render("│      ") + cCyan.Render("╰───╯") + strings.Repeat(" ", W-13) + cGray.Render("│")

	// Line 4: Label top border (width 74)
	l4 := cGray.Render("│  ") + cWhite.Render("╭"+strings.Repeat("─", innerLabelW)+"╮") + cGray.Render("  │")

	// Line 5: Label top space with Track Name (width 74)
	trackClean := truncateStr(track, innerLabelW-2)
	tw := lipgloss.Width(trackClean)
	tLeft := (innerLabelW - tw) / 2
	tRight := innerLabelW - tw - tLeft
	l5 := cGray.Render("│  ") + cWhite.Render("│") + strings.Repeat(" ", tLeft) + cTrack.Render(trackClean) + strings.Repeat(" ", tRight) + cWhite.Render("│") + cGray.Render("  │")

	// Lines 6-10: Spools and Center Tape Window (each line width 74)
	// Center window lines are exactly 15 visual characters
	centerWin := [5]string{
		"╭─────────────╮",
		fmt.Sprintf("│%s│ │ │%s│", leftTape, rightTape),
		fmt.Sprintf("│%s│ │ │%s│", leftTape, rightTape),
		fmt.Sprintf("│%s│ │ │%s│", leftTape, rightTape),
		"╰─────────────╯",
	}

	g1 := strings.Repeat(" ", 8)
	g2 := strings.Repeat(" ", 8)
	g3 := strings.Repeat(" ", 8)
	g4 := strings.Repeat(" ", 9)

	spoolLines := make([]string, 5)
	for i := 0; i < 5; i++ {
		rowContent := g1 + cCyan.Render(spoolL[i]) + g2 + cDark.Render(centerWin[i]) + g3 + cCyan.Render(spoolR[i]) + g4
		spoolLines[i] = cGray.Render("│  ") + cWhite.Render("│") + rowContent + cWhite.Render("│") + cGray.Render("  │")
	}

	// Line 11: Label divider (width 74)
	l11 := cGray.Render("│  ") + cWhite.Render("├"+strings.Repeat("─", innerLabelW)+"┤") + cGray.Render("  │")

	// Line 12: Bottom label info: Artist, Progress Bar / Time, Status (width 74)
	elapsed := formatDuration(c.playerStatus.CurrentMs)
	total := formatDuration(c.playerStatus.DurationMs)
	barW := 12
	filled := max(0, min(barW, int(prog*float64(barW))))
	barStr := strings.Repeat("█", filled) + strings.Repeat("─", barW-filled)

	var statusTag string
	if c.playerStatus.ShowVolume {
		statusTag = fmt.Sprintf("VOL [%s]", volumeBar(c.playerStatus.Volume, c.playerStatus.VolumeMax, 6))
	} else if c.playerStatus.Playing {
		if c.playerStatus.Shuffled {
			statusTag = "▶ PLAYING ⇌"
		} else {
			statusTag = "▶ PLAYING"
		}
	} else if c.playerStatus.Online {
		statusTag = "⏸ PAUSED"
	} else {
		statusTag = "● READY"
	}

	artistClean := truncateStr(artist, 20)
	plainMeta := fmt.Sprintf("%s • %s [%s] %s   %s", artistClean, elapsed, barStr, total, statusTag)
	pw := lipgloss.Width(plainMeta)
	if pw > innerLabelW {
		avail := innerLabelW - (pw - lipgloss.Width(artistClean))
		if avail < 4 {
			artistClean = ""
			plainMeta = fmt.Sprintf("%s [%s] %s   %s", elapsed, barStr, total, statusTag)
		} else {
			artistClean = truncateStr(artist, avail)
			plainMeta = fmt.Sprintf("%s • %s [%s] %s   %s", artistClean, elapsed, barStr, total, statusTag)
		}
		pw = lipgloss.Width(plainMeta)
	}

	mLeft := (innerLabelW - pw) / 2
	mRight := innerLabelW - pw - mLeft

	var metaContent string
	if artistClean != "" {
		metaContent = cArtist.Render(artistClean) + " • " + elapsed + " [" + cGreen.Render(barStr) + "] " + total + "   " + cStatus.Render(statusTag)
	} else {
		metaContent = elapsed + " [" + cGreen.Render(barStr) + "] " + total + "   " + cStatus.Render(statusTag)
	}

	l12 := cGray.Render("│  ") + cWhite.Render("│") + strings.Repeat(" ", mLeft) + metaContent + strings.Repeat(" ", mRight) + cWhite.Render("│") + cGray.Render("  │")

	// Line 13: Label bottom border (width 74)
	l13 := cGray.Render("│  ") + cWhite.Render("╰"+strings.Repeat("─", innerLabelW)+"╯") + cGray.Render("  │")

	// Line 14: Lower cassette body texture (width 74)
	l14 := cGray.Render("│" + strings.Repeat("░", W-2) + "│")

	// Line 15: Tape head opening - upper guide holes (width 74)
	l15 := cGray.Render("│    ╲   ") + cCyan.Render("( )") + strings.Repeat(" ", W-24) + cCyan.Render("( )") + cGray.Render("   ╱    │")

	// Line 16: Tape head opening - drive holes and bottom screws (width 74)
	l16 := cGray.Render("│") + cCyan.Render("(o)") + cGray.Render("  ╲         ") + cCyan.Render("( )") + strings.Repeat(" ", W-38) + cCyan.Render("( )") + cGray.Render("         ╱  ") + cCyan.Render("(o)") + cGray.Render("│")

	// Line 17: Cassette bottom edge (width 74)
	l17 := cGray.Render("╰──────╲" + strings.Repeat("─", W-16) + "╱──────╯")

	lines := []string{
		l0, l1, l2, l3, l4, l5,
		spoolLines[0], spoolLines[1], spoolLines[2], spoolLines[3], spoolLines[4],
		l11, l12, l13, l14, l15, l16, l17,
	}

	return strings.Join(lines, "\n")
}

func truncateStr(s string, maxLen int) string {
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

func volumeBar(vol, maxVol, width int) string {
	if maxVol <= 0 {
		maxVol = 100
	}
	ratio := float64(vol) / float64(maxVol)
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	filled := int(ratio * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("─", width-filled)
}
