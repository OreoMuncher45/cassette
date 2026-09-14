package settings

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSettingsAutostartOption(t *testing.T) {
	m := NewModel()
	foundAutostart := false
	for _, opt := range m.options {
		if opt.Kind == ItemAutostart {
			foundAutostart = true
			if opt.Label != "Autostart on Boot" {
				t.Errorf("unexpected label: %s", opt.Label)
			}
		}
	}
	if !foundAutostart {
		t.Errorf("expected ItemAutostart option in settings")
	}

	m.Open()
	if !m.IsOpen() {
		t.Errorf("expected model to be open")
	}

	// Move cursor to bottom where autostart is
	m.cursor = len(m.options) - 1
	// Toggle with Enter
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	view := m.View()
	if view == "" {
		t.Errorf("expected non-empty view")
	}
}
