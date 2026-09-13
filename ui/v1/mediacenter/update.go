package mediacenter

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.SearchFocused() {
			return m.mediaPanel.Update(msg)
		}

		switch {
		case key.Matches(msg, m.keys.ToggleLyrics):
			m.ToggleLyrics()
			return nil
		case key.Matches(msg, m.keys.ToggleQueue):
			m.ToggleQueue()
			return nil
		case key.Matches(msg, m.keys.TogglePanel):
			if msg.String() == "tab" {
				m.SwapLeftPanel()
			} else {
				m.ToggleLibrary()
			}
			return nil
		case key.Matches(msg, m.keys.ZenMode):
			m.zenMode = !m.zenMode
			return nil
		}

		if m.leftPanel != LeftPanelLibrary {
			return nil
		}
	}
	return m.mediaPanel.Update(msg)
}
