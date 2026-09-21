package lyrics

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cassette/core/logger"
)

// LyricWord represents a single word with per-word timing for karaoke rendering.
type LyricWord struct {
	StartMs int
	EndMs   int
	Text    string
}

// LyricLine represents a single lyrics line with optional per-word timing.
type LyricLine struct {
	TimeMs  int
	EndTime int
	Text    string
	Words   []LyricWord
}

// Lyrics holds the complete lyrics data for a track.
type Lyrics struct {
	TrackName    string
	ArtistName   string
	PlainLyrics  []string
	SyncedLyrics []LyricLine
	HasSynced    bool
	HasWordSync  bool
}

// Service manages lyrics fetching and caching.
type Service struct {
	client *http.Client
	cache  map[string]*Lyrics
	mu     sync.RWMutex
}

var (
	timeRegex      = regexp.MustCompile(`^\[(\d+):(\d+(?:\.\d+)?)\]\s*(.*)$`)
	enhancedLRCReg = regexp.MustCompile(`<(\d+):(\d+(?:\.\d+)?)>\s*([^<]*)`)
	defaultService *Service
	once           sync.Once
)

func GetService() *Service {
	once.Do(func() {
		dialer := &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: 3 * time.Second}
					c, err := d.DialContext(ctx, "udp", "8.8.8.8:53")
					if err != nil {
						return d.DialContext(ctx, "udp", "1.1.1.1:53")
					}
					return c, nil
				},
			},
			Timeout: 5 * time.Second,
		}

		transport := &http.Transport{
			DialContext: dialer.DialContext,
		}

		defaultService = &Service{
			client: &http.Client{
				Transport: transport,
				Timeout:   6 * time.Second,
			},
			cache: make(map[string]*Lyrics),
		}
	})
	return defaultService
}

// FetchLyrics attempts multiple providers in cascade order:
// 1. BetterLyrics (Apple Music TTML with per-word timing)
// 2. Portato (QQ Music word timing)
// 3. LRCLIB (line-synced LRC + plain text fallback)
func (s *Service) FetchLyrics(track, artist string) (*Lyrics, error) {
	track = strings.TrimSpace(track)
	artist = strings.TrimSpace(artist)
	if track == "" || artist == "" {
		return nil, fmt.Errorf("empty track or artist name")
	}

	key := strings.ToLower(artist + " - " + track)
	s.mu.RLock()
	if cached, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	cleanTrack := cleanTrackName(track)

	// 1. Try BetterLyrics (Apple Music TTML word-synced)
	if result, err := s.fetchBetterLyrics(cleanTrack, artist); err == nil && result != nil {
		logger.Log.Info().Str("track", track).Msg("word-synced lyrics from BetterLyrics")
		s.mu.Lock()
		s.cache[key] = result
		s.mu.Unlock()
		return result, nil
	}

	// 2. Try Portato (QQ Music word timing)
	if result, err := s.fetchPortato(cleanTrack, artist); err == nil && result != nil {
		logger.Log.Info().Str("track", track).Msg("word-synced lyrics from Portato/QQ")
		s.mu.Lock()
		s.cache[key] = result
		s.mu.Unlock()
		return result, nil
	}

	// 3. Fallback to LRCLIB (line-synced + plain text)
	result, err := s.fetchLRCLIB(cleanTrack, artist)
	if err != nil {
		logger.Log.Warn().Err(err).Str("track", track).Msg("all lyrics sources failed")
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = result
	s.mu.Unlock()

	return result, nil
}

// --- BetterLyrics (Apple Music TTML) ---

type betterLyricsResponse struct {
	Lyrics string `json:"lyrics"` // TTML XML string
}

func (s *Service) fetchBetterLyrics(track, artist string) (*Lyrics, error) {
	reqURL := fmt.Sprintf("https://lyrics-api.boidu.dev/getLyrics?title=%s&artist=%s",
		url.QueryEscape(track),
		url.QueryEscape(artist),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cassette-tui/2.0 (https://github.com/OreoMuncher45/cassette)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BetterLyrics returned status %d", resp.StatusCode)
	}

	var data betterLyricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Lyrics == "" {
		return nil, fmt.Errorf("empty TTML from BetterLyrics")
	}

	lines := parseTTML(data.Lyrics)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no parseable TTML lines")
	}

	hasWord := false
	for _, l := range lines {
		if len(l.Words) > 0 {
			hasWord = true
			break
		}
	}

	return &Lyrics{
		TrackName:    track,
		ArtistName:   artist,
		SyncedLyrics: lines,
		HasSynced:    true,
		HasWordSync:  hasWord,
	}, nil
}

// --- Portato (QQ Music) ---

type portatoResponse struct {
	Lyrics string `json:"lyrics"` // Enhanced LRC with word timing
}

func (s *Service) fetchPortato(track, artist string) (*Lyrics, error) {
	reqURL := fmt.Sprintf("https://lyrics-api.boidu.dev/qq/getLyrics?title=%s&artist=%s",
		url.QueryEscape(track),
		url.QueryEscape(artist),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cassette-tui/2.0 (https://github.com/OreoMuncher45/cassette)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Portato returned status %d", resp.StatusCode)
	}

	var data portatoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Lyrics == "" {
		return nil, fmt.Errorf("empty lyrics from Portato")
	}

	lines := parseEnhancedLRC(data.Lyrics)
	if len(lines) == 0 {
		// Try parsing as standard LRC
		lines = parseSyncedLyrics(data.Lyrics)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("no parseable lines from Portato")
	}

	hasWord := false
	for _, l := range lines {
		if len(l.Words) > 0 {
			hasWord = true
			break
		}
	}

	return &Lyrics{
		TrackName:    track,
		ArtistName:   artist,
		SyncedLyrics: lines,
		HasSynced:    true,
		HasWordSync:  hasWord,
	}, nil
}

// --- LRCLIB (line-synced fallback) ---

type lrclibResponse struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
	Duration     float64 `json:"duration"`
}

func (s *Service) fetchLRCLIB(track, artist string) (*Lyrics, error) {
	reqURL := fmt.Sprintf("https://lrclib.net/api/get?track_name=%s&artist_name=%s",
		url.QueryEscape(track),
		url.QueryEscape(artist),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cassette-tui/2.0 (https://github.com/OreoMuncher45/cassette)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lyrics not found (status %d)", resp.StatusCode)
	}

	var data lrclibResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	result := &Lyrics{
		TrackName:  data.TrackName,
		ArtistName: data.ArtistName,
	}

	if data.SyncedLyrics != "" {
		result.SyncedLyrics = parseSyncedLyrics(data.SyncedLyrics)
		result.HasSynced = len(result.SyncedLyrics) > 0
	}

	if data.PlainLyrics != "" {
		for _, l := range strings.Split(data.PlainLyrics, "\n") {
			l = strings.TrimSpace(l)
			if l != "" {
				result.PlainLyrics = append(result.PlainLyrics, l)
			}
		}
	}

	return result, nil
}

// --- Parsers ---

// parseSyncedLyrics parses standard LRC format: [mm:ss.xx] text
func parseSyncedLyrics(raw string) []LyricLine {
	var lines []LyricLine
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) == 4 {
			min, _ := strconv.Atoi(matches[1])
			sec, _ := strconv.ParseFloat(matches[2], 64)
			totalMs := int(float64(min*60)*1000 + sec*1000)
			text := strings.TrimSpace(matches[3])
			if text != "" {
				lines = append(lines, LyricLine{
					TimeMs: totalMs,
					Text:   text,
				})
			}
		}
	}
	return lines
}

// parseEnhancedLRC parses Enhanced LRC with inline word timing:
// [mm:ss.xx] <mm:ss.xx> word1 <mm:ss.xx> word2 ...
func parseEnhancedLRC(raw string) []LyricLine {
	var lines []LyricLine

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract line timestamp
		lineMatches := timeRegex.FindStringSubmatch(line)
		if len(lineMatches) < 4 {
			continue
		}

		min, _ := strconv.Atoi(lineMatches[1])
		sec, _ := strconv.ParseFloat(lineMatches[2], 64)
		lineMs := int(float64(min*60)*1000 + sec*1000)
		content := lineMatches[3]

		// Try to extract inline word timestamps
		wordMatches := enhancedLRCReg.FindAllStringSubmatch(content, -1)
		if len(wordMatches) > 0 {
			lyricLine := LyricLine{
				TimeMs: lineMs,
			}
			var textParts []string
			for i, wm := range wordMatches {
				if len(wm) < 4 {
					continue
				}
				wMin, _ := strconv.Atoi(wm[1])
				wSec, _ := strconv.ParseFloat(wm[2], 64)
				wordStartMs := int(float64(wMin*60)*1000 + wSec*1000)
				wordText := strings.TrimSpace(wm[3])
				if wordText == "" {
					continue
				}

				wordEndMs := 0
				if i+1 < len(wordMatches) && len(wordMatches[i+1]) >= 3 {
					nextMin, _ := strconv.Atoi(wordMatches[i+1][1])
					nextSec, _ := strconv.ParseFloat(wordMatches[i+1][2], 64)
					wordEndMs = int(float64(nextMin*60)*1000 + nextSec*1000)
				} else {
					wordEndMs = wordStartMs + 500 // Default word duration
				}

				lyricLine.Words = append(lyricLine.Words, LyricWord{
					StartMs: wordStartMs,
					EndMs:   wordEndMs,
					Text:    wordText,
				})
				textParts = append(textParts, wordText)
			}

			lyricLine.Text = strings.Join(textParts, " ")
			if len(lyricLine.Words) > 0 {
				lyricLine.EndTime = lyricLine.Words[len(lyricLine.Words)-1].EndMs
			}
			lines = append(lines, lyricLine)
		} else {
			// No inline word timestamps, treat as regular synced line
			text := strings.TrimSpace(content)
			if text != "" {
				lines = append(lines, LyricLine{
					TimeMs: lineMs,
					Text:   text,
				})
			}
		}
	}

	return lines
}

// parseTTML parses Apple Music TTML (Timed Text Markup Language) with per-word span timing.
// Format: <p begin="00:00:05.000" end="00:00:10.000"><span begin="..." end="...">word</span>...</p>
func parseTTML(ttmlStr string) []LyricLine {
	var lines []LyricLine

	// Parse the TTML XML
	type TTMLSpan struct {
		XMLName xml.Name `xml:"span"`
		Begin   string   `xml:"begin,attr"`
		End     string   `xml:"end,attr"`
		Text    string   `xml:",chardata"`
	}

	type TTMLParagraph struct {
		XMLName xml.Name   `xml:"p"`
		Begin   string     `xml:"begin,attr"`
		End     string     `xml:"end,attr"`
		Spans   []TTMLSpan `xml:"span"`
		Text    string     `xml:",chardata"`
	}

	type TTMLDiv struct {
		Paragraphs []TTMLParagraph `xml:"p"`
	}

	type TTMLBody struct {
		Divs []TTMLDiv `xml:"div"`
	}

	type TTML struct {
		XMLName xml.Name `xml:"tt"`
		Body    TTMLBody `xml:"body"`
	}

	var ttml TTML
	if err := xml.Unmarshal([]byte(ttmlStr), &ttml); err != nil {
		// Try fallback: look for <p> elements manually
		return parseTTMLFallback(ttmlStr)
	}

	for _, div := range ttml.Body.Divs {
		for _, p := range div.Paragraphs {
			lineStartMs := parseTTMLTime(p.Begin)
			lineEndMs := parseTTMLTime(p.End)

			lyricLine := LyricLine{
				TimeMs:  lineStartMs,
				EndTime: lineEndMs,
			}

			if len(p.Spans) > 0 {
				var textParts []string
				for _, span := range p.Spans {
					wordText := strings.TrimSpace(span.Text)
					if wordText == "" {
						continue
					}
					wordStart := parseTTMLTime(span.Begin)
					wordEnd := parseTTMLTime(span.End)
					lyricLine.Words = append(lyricLine.Words, LyricWord{
						StartMs: wordStart,
						EndMs:   wordEnd,
						Text:    wordText,
					})
					textParts = append(textParts, wordText)
				}
				lyricLine.Text = strings.Join(textParts, " ")
			} else {
				lyricLine.Text = strings.TrimSpace(p.Text)
			}

			if lyricLine.Text != "" {
				lines = append(lines, lyricLine)
			}
		}
	}

	return lines
}

// parseTTMLFallback uses regex-based parsing as a fallback for malformed TTML.
func parseTTMLFallback(ttmlStr string) []LyricLine {
	var lines []LyricLine

	pRegex := regexp.MustCompile(`<p[^>]*begin="([^"]*)"[^>]*end="([^"]*)"[^>]*>(.*?)</p>`)
	spanRegex := regexp.MustCompile(`<span[^>]*begin="([^"]*)"[^>]*end="([^"]*)"[^>]*>([^<]*)</span>`)

	pMatches := pRegex.FindAllStringSubmatch(ttmlStr, -1)
	for _, pm := range pMatches {
		if len(pm) < 4 {
			continue
		}

		lineStartMs := parseTTMLTime(pm[1])
		lineEndMs := parseTTMLTime(pm[2])
		content := pm[3]

		lyricLine := LyricLine{
			TimeMs:  lineStartMs,
			EndTime: lineEndMs,
		}

		spanMatches := spanRegex.FindAllStringSubmatch(content, -1)
		if len(spanMatches) > 0 {
			var textParts []string
			for _, sm := range spanMatches {
				if len(sm) < 4 {
					continue
				}
				wordText := strings.TrimSpace(sm[3])
				if wordText == "" {
					continue
				}
				wordStart := parseTTMLTime(sm[1])
				wordEnd := parseTTMLTime(sm[2])
				lyricLine.Words = append(lyricLine.Words, LyricWord{
					StartMs: wordStart,
					EndMs:   wordEnd,
					Text:    wordText,
				})
				textParts = append(textParts, wordText)
			}
			lyricLine.Text = strings.Join(textParts, " ")
		} else {
			// Strip any remaining tags
			tagStrip := regexp.MustCompile(`<[^>]+>`)
			lyricLine.Text = strings.TrimSpace(tagStrip.ReplaceAllString(content, " "))
		}

		if lyricLine.Text != "" {
			lines = append(lines, lyricLine)
		}
	}

	return lines
}

// parseTTMLTime parses TTML time format: "00:01:23.456" or "01:23.456" → milliseconds
func parseTTMLTime(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 3:
		// HH:MM:SS.mmm
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		sec, _ := strconv.ParseFloat(parts[2], 64)
		return h*3600000 + m*60000 + int(sec*1000)
	case 2:
		// MM:SS.mmm
		m, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.ParseFloat(parts[1], 64)
		return m*60000 + int(sec*1000)
	default:
		// Just seconds
		sec, _ := strconv.ParseFloat(s, 64)
		return int(sec * 1000)
	}
}

func cleanTrackName(name string) string {
	// Cut off at " - " (e.g. "Song - Remastered 2011")
	if idx := strings.Index(name, " - "); idx != -1 {
		name = name[:idx]
	}
	// Cut off parentheses if they contain remaster/live/version
	low := strings.ToLower(name)
	for _, kw := range []string{"(remaster", "(live", "(deluxe", "(bonus", "(feat", "(version"} {
		if idx := strings.Index(low, kw); idx != -1 {
			name = strings.TrimSpace(name[:idx])
			break
		}
	}
	return strings.TrimSpace(name)
}
