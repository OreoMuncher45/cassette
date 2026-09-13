package displayscreen

import (
	"testing"

	ansi "github.com/charmbracelet/x/ansi"
	"cassette/ui/v1/common"
)

func TestScrollTextDoesNotGoBlankAcrossTicks(t *testing.T) {
	m := NewModel()
	m.SetDisplayFromSong(common.SongInfo{
		Title:  "A Very Long Track Title",
		Artist: "A Verbose Artist",
		Album:  "An Even Longer Album Name",
	})

	const width = 12
	frames := len(m.display) * 2
	if frames == 0 {
		t.Fatal("expected non-empty display")
	}

	for i := 0; i < frames; i++ {
		frame := m.scrollText(m.display, width)
		if ansi.StringWidth(frame) == 0 {
			t.Fatalf("blank frame at tick %d with offset %d", i, m.scrollOffset)
		}
		m.NextFrame()
	}
}
