package lyrics

import (
	"testing"
)

func TestParseSyncedLyrics(t *testing.T) {
	raw := `[00:01.00] First line
[00:05.50] Second line
[01:00.00] Third line`

	lines := parseSyncedLyrics(raw)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].TimeMs != 1000 || lines[0].Text != "First line" {
		t.Errorf("line 0: got %d/%s", lines[0].TimeMs, lines[0].Text)
	}
	if lines[1].TimeMs != 5500 || lines[1].Text != "Second line" {
		t.Errorf("line 1: got %d/%s", lines[1].TimeMs, lines[1].Text)
	}
	if lines[2].TimeMs != 60000 || lines[2].Text != "Third line" {
		t.Errorf("line 2: got %d/%s", lines[2].TimeMs, lines[2].Text)
	}
}

func TestParseTTMLTime(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"00:01:23.456", 83456},
		{"01:23.456", 83456},
		{"23.456", 23456},
		{"00:00:05.000", 5000},
		{"", 0},
	}

	for _, tc := range tests {
		result := parseTTMLTime(tc.input)
		if result != tc.expected {
			t.Errorf("parseTTMLTime(%q) = %d, want %d", tc.input, result, tc.expected)
		}
	}
}

func TestParseTTML(t *testing.T) {
	ttml := `<?xml version="1.0" encoding="utf-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body>
    <div>
      <p begin="00:00:05.000" end="00:00:10.000">
        <span begin="00:00:05.000" end="00:00:06.500">Hello</span>
        <span begin="00:00:06.500" end="00:00:08.000">World</span>
        <span begin="00:00:08.000" end="00:00:10.000">Test</span>
      </p>
    </div>
  </body>
</tt>`

	lines := parseTTML(ttml)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if len(lines[0].Words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(lines[0].Words))
	}
	if lines[0].Words[0].Text != "Hello" || lines[0].Words[0].StartMs != 5000 {
		t.Errorf("word 0: %+v", lines[0].Words[0])
	}
	if lines[0].Words[1].Text != "World" || lines[0].Words[1].StartMs != 6500 {
		t.Errorf("word 1: %+v", lines[0].Words[1])
	}
	if lines[0].Text != "Hello World Test" {
		t.Errorf("expected 'Hello World Test', got '%s'", lines[0].Text)
	}
}

func TestParseEnhancedLRC(t *testing.T) {
	raw := `[00:05.00] <00:05.00> Hello <00:06.50> World <00:08.00> Test`

	lines := parseEnhancedLRC(raw)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if len(lines[0].Words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(lines[0].Words))
	}
	if lines[0].Words[0].Text != "Hello" || lines[0].Words[0].StartMs != 5000 {
		t.Errorf("word 0: %+v", lines[0].Words[0])
	}
	if lines[0].Words[1].Text != "World" || lines[0].Words[1].StartMs != 6500 {
		t.Errorf("word 1: %+v", lines[0].Words[1])
	}
}

func TestCleanTrackName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Song - Remastered 2011", "Song"},
		{"Song (Remastered 2021)", "Song"},
		{"Song (Live at Wembley)", "Song"},
		{"Song (feat. Artist)", "Song"},
		{"Normal Song", "Normal Song"},
	}
	for _, tc := range tests {
		got := cleanTrackName(tc.input)
		if got != tc.expected {
			t.Errorf("cleanTrackName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
