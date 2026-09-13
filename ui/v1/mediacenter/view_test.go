package mediacenter

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"cassette/ui/v1/common"
)

func TestViewHidesPlayerWhenPanelIsOpenAndViewportIsTooSmall(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())

	model.Update(tea.KeyPressMsg(tea.Key{Text: "P", Code: 'P'}))

	view := model.View(40, 24)

	if strings.Contains(view, "CASSETTE") {
		t.Fatalf("view = %q, did not expect player to render in narrow viewport", view)
	}
}

func TestViewKeepsPlayerWhenPanelIsOpenAndViewportFits(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())

	model.Update(tea.KeyPressMsg(tea.Key{Text: "P", Code: 'P'}))

	view := model.View(120, 40)

	if !strings.Contains(view, "CASSETTE") {
		t.Fatalf("view = %q, expected player to render in wide viewport", view)
	}
}

func TestViewShowsPlayerAgainAfterClosingLibrary(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())

	model.Update(tea.KeyPressMsg(tea.Key{Text: "P", Code: 'P'}))
	model.CloseLibrary()

	view := model.View(120, 40)

	if !strings.Contains(view, "CASSETTE") {
		t.Fatalf("view = %q, expected player to render after closing library", view)
	}
}

func TestViewDoesNotHidePlayerWhenViewportIsTooSmallAndPanelIsClosed(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())

	view := model.View(1, 1)

	if !strings.Contains(view, "CASSETTE") {
		t.Fatalf("view = %q, expected player to remain rendered when panel is closed", view)
	}
}

func TestViewHidesDisplayAndControlsInZenMode(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())

	model.Update(tea.KeyPressMsg(tea.Key{Text: "z", Code: 'z'}))

	view := model.View(120, 40)

	if !strings.Contains(view, "CASSETTE") {
		t.Fatalf("view = %q, expected cassette to remain visible in zen mode", view)
	}
	if strings.Contains(view, "Cassette: The retro terminal Spotify player") {
		t.Fatalf("view = %q, did not expect display screen content in zen mode", view)
	}
	for _, icon := range []string{"|<", "<<", "|>", "||", ">>", ">|"} {
		if strings.Contains(view, icon) {
			t.Fatalf("view = %q, did not expect control icon %q in zen mode", view, icon)
		}
	}
}

func TestUpdateTogglesZenMode(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())

	model.Update(tea.KeyPressMsg(tea.Key{Text: "z", Code: 'z'}))
	if !model.IsZenMode() {
		t.Fatal("expected zen mode to be enabled after pressing z")
	}

	model.Update(tea.KeyPressMsg(tea.Key{Text: "z", Code: 'z'}))
	if model.IsZenMode() {
		t.Fatal("expected zen mode to be disabled after pressing z again")
	}
}
