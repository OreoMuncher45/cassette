package player

import (
	"strings"
	"testing"
)

func TestViewHidesControlsInZenMode(t *testing.T) {
	model := NewModel()

	view := model.View(true)

	if !strings.Contains(view, "CASSETTE") {
		t.Fatalf("view = %q, expected cassette to render in zen mode", view)
	}
	for _, icon := range []string{"|<", "<<", "|>", "||", ">>", ">|"} {
		if strings.Contains(view, icon) {
			t.Fatalf("view = %q, did not expect control icon %q in zen mode", view, icon)
		}
	}
}

func TestViewShowsControlsOutsideZenMode(t *testing.T) {
	model := NewModel()

	view := model.View(false)

	for _, icon := range []string{"|<", "<<", "|>", "||", ">>", ">|"} {
		if !strings.Contains(view, icon) {
			t.Fatalf("view = %q, expected control icon %q outside zen mode", view, icon)
		}
	}
}
