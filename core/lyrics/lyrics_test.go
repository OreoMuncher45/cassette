package lyrics

import (
	"testing"
)

func TestCleanTrackName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Bohemian Rhapsody - Remastered 2011", "Bohemian Rhapsody"},
		{"Hotel California (Remastered)", "Hotel California"},
		{"Comfortably Numb - Live at Pompeii", "Comfortably Numb"},
		{"Stan (feat. Dido)", "Stan"},
		{"Billie Jean", "Billie Jean"},
	}

	for _, c := range cases {
		got := cleanTrackName(c.input)
		if got != c.expected {
			t.Errorf("cleanTrackName(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestParseSyncedLyrics(t *testing.T) {
	raw := `[00:12.34] First line of song
[00:25.80] Second line of song
[01:05.10] Chorus line`

	lines := parseSyncedLyrics(raw)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	if lines[0].TimeMs != 12340 || lines[0].Text != "First line of song" {
		t.Errorf("line 0 mismatch: %+v", lines[0])
	}
	if lines[1].TimeMs != 25800 || lines[1].Text != "Second line of song" {
		t.Errorf("line 1 mismatch: %+v", lines[1])
	}
	if lines[2].TimeMs != 65100 || lines[2].Text != "Chorus line" {
		t.Errorf("line 2 mismatch: %+v", lines[2])
	}
}
