package lyrics

import (
	"context"
	"encoding/json"
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

type LyricLine struct {
	TimeMs int
	Text   string
}

type Lyrics struct {
	TrackName    string
	ArtistName   string
	PlainLyrics  []string
	SyncedLyrics []LyricLine
	HasSynced    bool
}

type Service struct {
	client *http.Client
	cache  map[string]*Lyrics
	mu     sync.RWMutex
}

var (
	timeRegex      = regexp.MustCompile(`^\[(\d+):(\d+(?:\.\d+)?)\]\s*(.*)$`)
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

type lrclibResponse struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
	Duration     float64 `json:"duration"`
}

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

	// Strip common track extras like "(Remastered 2021)" or "- Live"
	cleanTrack := cleanTrackName(track)

	reqURL := fmt.Sprintf("https://lrclib.net/api/get?track_name=%s&artist_name=%s",
		url.QueryEscape(cleanTrack),
		url.QueryEscape(artist),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cassette-tui/1.0 (https://github.com/OreoMuncher45/cassette)")

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Log.Warn().Err(err).Str("track", track).Msg("failed to fetch lyrics from lrclib")
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

	s.mu.Lock()
	s.cache[key] = result
	s.mu.Unlock()

	return result, nil
}

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
