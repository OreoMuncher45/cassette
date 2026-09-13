package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestViewHidesHelpLineInZenMode(t *testing.T) {
	model := NewModel()
	model.setSize(120, 40)
	model.mediaCenter.Update(tea.KeyPressMsg(tea.Key{Text: "z", Code: 'z'}))

	view := model.View()

	if !strings.Contains(view.Content, "CASSETTE") {
		t.Fatalf("view = %q, expected player to render in zen mode", view.Content)
	}
	if strings.Contains(view.Content, "toggle help") {
		t.Fatalf("view = %q, did not expect help line in zen mode", view.Content)
	}
}
