package keybinds

import (
	"fmt"
	"strings"

	"cassette/core/theme"
	"cassette/ui/v1/common"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	keys           *common.AppKeyMap
	actions        []common.RebindAction
	filtered       []common.RebindAction
	cursor         int
	filter         string
	rebinding      bool
	rebindActionID string
	width          int
	height         int
	statusMsg      string
	isOpen         bool
}

func NewModel(keys *common.AppKeyMap) Model {
	m := Model{
		keys:   keys,
		width:  80,
		height: 24,
	}
	m.Refresh()
	return m
}

func (m *Model) Open() {
	m.isOpen = true
	m.filter = ""
	m.rebinding = false
	m.statusMsg = ""
	m.Refresh()
}

func (m *Model) Close() {
	m.isOpen = false
	m.rebinding = false
	m.filter = ""
}

func (m *Model) IsOpen() bool {
	return m.isOpen
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) Refresh() {
	if m.keys == nil {
		return
	}
	m.actions = m.keys.GetRebindActions()
	m.applyFilter()
}

func (m *Model) applyFilter() {
	if strings.TrimSpace(m.filter) == "" {
		m.filtered = m.actions
	} else {
		query := strings.ToLower(strings.TrimSpace(m.filter))
		var res []common.RebindAction
		for _, a := range m.actions {
			if strings.Contains(strings.ToLower(a.Name), query) ||
				strings.Contains(strings.ToLower(a.Description), query) ||
				strings.Contains(strings.ToLower(strings.Join(a.Keys, " ")), query) {
				res = append(res, a)
			}
		}
		m.filtered = res
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !m.isOpen {
		return nil, false
	}

	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}

	keyStr := keyMsg.String()

	// 1. In Rebinding capture mode
	if m.rebinding {
		if keyStr == "esc" {
			m.rebinding = false
			m.statusMsg = "Rebinding cancelled."
			return nil, true
		}

		// Clean key string
		assignedKey := keyStr
		if assignedKey == " " {
			assignedKey = "space"
		}

		m.keys.SetActionKey(m.rebindActionID, assignedKey)
		m.rebinding = false
		m.statusMsg = fmt.Sprintf("✓ Rebound to [%s]", assignedKey)
		m.Refresh()
		return nil, true
	}

	// 2. Normal mode
	switch keyStr {
	case "f1":
		m.Close()
		return nil, true

	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.applyFilter()
			return nil, true
		}
		m.Close()
		return nil, true

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return nil, true

	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return nil, true

	case "enter":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			action := m.filtered[m.cursor]
			m.rebinding = true
			m.rebindActionID = action.ID
			m.statusMsg = fmt.Sprintf("Press NEW key for '%s'...", action.Name)
		}
		return nil, true

	case "ctrl+r":
		m.keys.ResetDefaults()
		m.Refresh()
		m.statusMsg = "✓ Reset all keybinds to defaults."
		return nil, true

	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
		return nil, true

	default:
		// Filter search input
		if len(keyStr) == 1 && keyStr >= " " && keyStr <= "~" {
			m.filter += keyStr
			m.applyFilter()
			return nil, true
		}
	}

	return nil, true
}

func (m *Model) View() string {
	if !m.isOpen {
		return ""
	}

	th := theme.Get()
	cPrim := th.PrimaryColor()
	cSec := th.SecondaryColor()
	cBorder := th.BorderColor()

	boxW := min(78, max(50, m.width-4))
	boxH := min(26, max(16, m.height-4))

	stBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cBorder).
		Width(boxW).
		Height(boxH).
		Padding(0, 1)

	stTitle := lipgloss.NewStyle().Foreground(cPrim).Bold(true)
	stMeta := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	stFilter := lipgloss.NewStyle().Foreground(cSec).Bold(true)
	stHeader := lipgloss.NewStyle().Foreground(cPrim).Bold(true)
	stSelected := lipgloss.NewStyle().Background(cPrim).Foreground(lipgloss.Color("0")).Bold(true)
	stKey := lipgloss.NewStyle().Foreground(cSec).Bold(true)
	stDesc := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	titleLine := stTitle.Render("⌨  CASSETTE KEYBINDINGS MANAGER (F1)")

	searchBar := stMeta.Render("Search: ") + stFilter.Render("["+m.filter+"█]") + stMeta.Render(" (Type to filter, Esc to close)")

	headerLine := fmt.Sprintf("%-22s %-16s %s",
		stHeader.Render("ACTION"),
		stHeader.Render("KEY(S)"),
		stHeader.Render("DESCRIPTION"),
	)
	divider := stMeta.Render(strings.Repeat("─", boxW-4))

	// List items
	maxItems := boxH - 8
	if maxItems < 4 {
		maxItems = 4
	}

	startIdx := 0
	if m.cursor >= maxItems {
		startIdx = m.cursor - maxItems + 1
	}
	endIdx := min(len(m.filtered), startIdx+maxItems)

	var listLines []string
	if len(m.filtered) == 0 {
		listLines = append(listLines, stMeta.Render("  No matching actions found. Press backspace to clear search."))
	} else {
		for i := startIdx; i < endIdx; i++ {
			a := m.filtered[i]
			keyDisplay := strings.Join(a.Keys, ", ")
			if keyDisplay == "" {
				keyDisplay = "(unbound)"
			}

			line := fmt.Sprintf("%-22s %-16s %s",
				truncateStr(a.Name, 21),
				truncateStr(keyDisplay, 15),
				truncateStr(a.Description, boxW-44),
			)

			if i == m.cursor {
				if m.rebinding {
					line = lipgloss.NewStyle().Background(cSec).Foreground(lipgloss.Color("0")).Bold(true).Render("▶ " + line)
				} else {
					line = stSelected.Render("▶ " + line)
				}
			} else {
				line = "  " + fmt.Sprintf("%-22s %-16s %s",
					a.Name,
					stKey.Render(truncateStr(keyDisplay, 15)),
					stDesc.Render(truncateStr(a.Description, boxW-44)),
				)
			}
			listLines = append(listLines, line)
		}
	}

	for len(listLines) < maxItems {
		listLines = append(listLines, "")
	}

	// Status & Footer
	statusLine := m.statusMsg
	if m.rebinding {
		statusLine = lipgloss.NewStyle().Foreground(cSec).Bold(true).Render(m.statusMsg)
	} else if statusLine != "" {
		statusLine = lipgloss.NewStyle().Foreground(cPrim).Render(statusLine)
	} else {
		statusLine = stMeta.Render(fmt.Sprintf("%d actions available", len(m.filtered)))
	}

	footer := stMeta.Render("[Enter] Rebind key   [Ctrl+R] Reset Defaults   [↑/↓] Navigate   [Esc/F1] Close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		searchBar,
		divider,
		headerLine,
		strings.Join(listLines, "\n"),
		divider,
		statusLine,
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
