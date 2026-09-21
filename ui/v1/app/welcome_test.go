package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestKeyPressMsgString(t *testing.T) {
	keys := []tea.Key{
		{Code: tea.KeyEnter},
		{Code: tea.KeyEscape},
		{Code: tea.KeySpace},
		{Text: " ", Code: ' '},
		{Code: '\r'},
		{Code: '\n'},
		{Text: "q", Code: 'q'},
	}
	for _, k := range keys {
		msg := tea.KeyPressMsg(k)
		t.Logf("Key code=%d text=%q String()=%q Keystroke()=%q", k.Code, k.Text, msg.String(), msg.Keystroke())
	}
}

func TestDismissWelcomeModal(t *testing.T) {
	testKeys := []tea.Key{
		{Code: tea.KeySpace},
		{Code: tea.KeyEnter},
		{Code: tea.KeyEscape},
		{Text: "q", Code: 'q'},
		{Text: "x", Code: 'x'},
		{Text: " ", Code: ' '},
		{Code: '\r'},
		{Code: '\n'},
	}

	for _, k := range testKeys {
		model := NewModel()
		model.welcomeModel = &WelcomeModel{open: true}
		if !model.welcomeModel.IsOpen() {
			t.Fatalf("expected welcome model open before key %v", k)
		}
		model.Update(tea.KeyPressMsg(k))
		if model.welcomeModel.IsOpen() {
			t.Fatalf("expected welcome model to close on key %v", k)
		}
	}

	// Test WindowSizeMsg preserves open modal
	model := NewModel()
	model.welcomeModel = &WelcomeModel{open: true}
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if !model.welcomeModel.IsOpen() {
		t.Fatal("expected welcome model to remain open after WindowSizeMsg")
	}

	// Test ctrl+c returns quit
	_, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if cmd == nil {
		t.Fatal("expected quit cmd on ctrl+c")
	}
}

func TestWelcomeModelViewBorderAlignment(t *testing.T) {
	w := &WelcomeModel{open: true}
	view := w.View()
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("expected non-empty view")
	}
	expectedWidth := -1
	for i, l := range lines {
		width := lipgloss.Width(l)
		if expectedWidth == -1 {
			expectedWidth = width
		}
		if width != expectedWidth {
			t.Errorf("line %d has width %d, expected %d: %q", i, width, expectedWidth, l)
		}
	}
}


