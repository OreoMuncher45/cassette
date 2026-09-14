package settings

import (
	"fmt"
	"strings"

	"cassette/core/theme"
	"cassette/core/utils"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ItemKind int

const (
	ItemScheme ItemKind = iota
	ItemEffect
	ItemSpeed
	ItemAutostart
)

type SettingOption struct {
	Kind        ItemKind
	ID          string
	Label       string
	Description string
}

type Model struct {
	options []SettingOption
	cursor  int
	width   int
	height  int
	isOpen  bool
}

func NewModel() Model {
	options := []SettingOption{
		// Schemes
		{Kind: ItemScheme, ID: string(theme.SchemeAlbumArt), Label: "Album Art Reactive", Description: "Dynamic palette extracted from current song's album art"},
		{Kind: ItemScheme, ID: string(theme.SchemeRetroCyan), Label: "Retro Cassette Cyan", Description: "Classic Hi-Fi turquoise & warm amber HUD"},
		{Kind: ItemScheme, ID: string(theme.SchemeCyberpunk), Label: "Cyberpunk Neon", Description: "Hot neon pink & electric cyan night drive"},
		{Kind: ItemScheme, ID: string(theme.SchemeSynthwave), Label: "Synthwave 80s", Description: "Retro sunset purple & laser neon orange"},
		{Kind: ItemScheme, ID: string(theme.SchemeMatrix), Label: "Matrix Phosphor", Description: "Green phosphor CRT hacker terminal"},
		{Kind: ItemScheme, ID: string(theme.SchemeDracula), Label: "Dracula Dark", Description: "Vampire purple, vibrant pink & soft cyan"},
		{Kind: ItemScheme, ID: string(theme.SchemeNord), Label: "Nordic Frost", Description: "Arctic ice blue & calm polar slate"},
		{Kind: ItemScheme, ID: string(theme.SchemeAmber), Label: "Monochrome Amber", Description: "Warm amber vintage cassette deck display"},
		{Kind: ItemScheme, ID: string(theme.SchemeTokyoNight), Label: "Tokyo Night", Description: "Deep indigo midnight & pastel magenta"},
		{Kind: ItemScheme, ID: string(theme.SchemeSolarized), Label: "Solarized Dark", Description: "Balanced teal cyan & solar amber gold"},
		{Kind: ItemScheme, ID: string(theme.SchemePastel), Label: "Pastel Dream", Description: "Soft pastel pink & dreamy sky blue gradient"},

		// Effects
		{Kind: ItemEffect, ID: string(theme.EffectBreathing), Label: "Breathing Glow", Description: "Sine-wave pulsing glow on cassette spools & borders"},
		{Kind: ItemEffect, ID: string(theme.EffectRainbow), Label: "Rainbow RGB Cycle", Description: "Smooth flowing 360° rainbow spectrum cycle"},
		{Kind: ItemEffect, ID: string(theme.EffectPulse), Label: "Heartbeat Pulse", Description: "Rhythmic bass heartbeat pulse accent"},
		{Kind: ItemEffect, ID: string(theme.EffectStatic), Label: "Static Colors", Description: "Clean, distraction-free solid theme colors"},

		// Speeds
		{Kind: ItemSpeed, ID: string(theme.SpeedSlow), Label: "Speed: Chill (Slow)", Description: "Ambient, calm, slow visual pulsing"},
		{Kind: ItemSpeed, ID: string(theme.SpeedMedium), Label: "Speed: Groove (Medium)", Description: "Balanced, rhythmic visual transitions"},
		{Kind: ItemSpeed, ID: string(theme.SpeedFast), Label: "Speed: Hyper (Fast)", Description: "High-energy rave RGB visual cycling"},
		{Kind: ItemSpeed, ID: string(theme.SpeedUltra), Label: "Speed: Ultra (Ludicrous)", Description: "Maximum-velocity dynamic color shifting"},

		// System
		{Kind: ItemAutostart, ID: "autostart", Label: "Autostart on Boot", Description: "Launch Cassette automatically when logging into your desktop"},
	}

	return Model{
		options: options,
		width:   80,
		height:  26,
	}
}

func (m *Model) Open() {
	m.isOpen = true
}

func (m *Model) Close() {
	m.isOpen = false
}

func (m *Model) IsOpen() bool {
	return m.isOpen
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !m.isOpen {
		return nil, false
	}

	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}

	switch keyMsg.String() {
	case "esc", "f2":
		m.Close()
		return nil, true

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return nil, true

	case "down", "j":
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
		return nil, true

	case "enter", " ":
		if m.cursor >= 0 && m.cursor < len(m.options) {
			opt := m.options[m.cursor]
			th := theme.Get()
			switch opt.Kind {
			case ItemScheme:
				th.SetScheme(theme.SchemeID(opt.ID))
			case ItemEffect:
				th.SetEffect(theme.EffectID(opt.ID))
			case ItemSpeed:
				th.SetSpeed(theme.SpeedID(opt.ID))
			case ItemAutostart:
				_, _ = utils.ToggleAutostart()
			}
		}
		return nil, true
	}

	return nil, true
}

func (m *Model) View() string {
	if !m.isOpen {
		return ""
	}

	th := theme.Get()
	settings := th.GetSettings()
	cPrim := th.PrimaryColor()
	cSec := th.SecondaryColor()
	cBorder := th.BorderColor()

	boxW := min(80, max(52, m.width-4))
	boxH := min(28, max(18, m.height-4))

	stBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cBorder).
		Width(boxW).
		Height(boxH).
		Padding(0, 1)

	stTitle := lipgloss.NewStyle().Foreground(cPrim).Bold(true)
	stSection := lipgloss.NewStyle().Foreground(cSec).Bold(true)
	stMeta := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	stDesc := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	stActiveRadio := lipgloss.NewStyle().Foreground(cPrim).Bold(true)
	stSelected := lipgloss.NewStyle().Background(cPrim).Foreground(lipgloss.Color("0")).Bold(true)

	titleLine := stTitle.Render("⚙  CASSETTE VISUAL SETTINGS & THEMES (F2)")
	subTitle := stMeta.Render("Customize color schemes, dynamic album art palette, and RGB effects")
	divider := stMeta.Render(strings.Repeat("─", boxW-4))

	// Render items grouped by section
	maxDisplay := boxH - 7
	if maxDisplay < 6 {
		maxDisplay = 6
	}

	startIdx := 0
	if m.cursor >= maxDisplay {
		startIdx = m.cursor - maxDisplay + 1
	}
	endIdx := min(len(m.options), startIdx+maxDisplay)

	var listLines []string
	lastKind := ItemKind(-1)

	for i := startIdx; i < endIdx; i++ {
		opt := m.options[i]

		// Add section header if first item of that kind in display
		if opt.Kind != lastKind {
			switch opt.Kind {
			case ItemScheme:
				listLines = append(listLines, stSection.Render("── COLOR SCHEMES ──────────────────────────────────────"))
			case ItemEffect:
				listLines = append(listLines, stSection.Render("── VISUAL EFFECTS ─────────────────────────────────────"))
			case ItemSpeed:
				listLines = append(listLines, stSection.Render("── ANIMATION SPEED ────────────────────────────────────"))
			case ItemAutostart:
				listLines = append(listLines, stSection.Render("── SYSTEM INTEGRATION ─────────────────────────────────"))
			}
			lastKind = opt.Kind
		}

		isSelectedActive := false
		switch opt.Kind {
		case ItemScheme:
			isSelectedActive = (settings.Scheme == theme.SchemeID(opt.ID))
		case ItemEffect:
			isSelectedActive = (settings.Effect == theme.EffectID(opt.ID))
		case ItemSpeed:
			isSelectedActive = (settings.Speed == theme.SpeedID(opt.ID))
		case ItemAutostart:
			isSelectedActive = utils.IsAutostartEnabled()
		}

		radio := "[ ]"
		if isSelectedActive {
			if opt.Kind == ItemAutostart {
				radio = stActiveRadio.Render("[✔]")
			} else {
				radio = stActiveRadio.Render("[●]")
			}
		}

		descW := boxW - 32
		if descW < 10 {
			descW = 10
		}

		itemText := fmt.Sprintf("%s %-22s %s", radio, opt.Label, stDesc.Render(truncateStr(opt.Description, descW)))

		if i == m.cursor {
			listLines = append(listLines, stSelected.Render("▶ "+itemText))
		} else {
			listLines = append(listLines, "  "+itemText)
		}
	}

	for len(listLines) < maxDisplay {
		listLines = append(listLines, "")
	}

	footer := stMeta.Render("[↑/↓] Navigate   [Enter/Space] Select   [Esc/F2] Save & Close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		subTitle,
		divider,
		strings.Join(listLines, "\n"),
		divider,
		footer,
	)

	return stBorder.Render(content)
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
