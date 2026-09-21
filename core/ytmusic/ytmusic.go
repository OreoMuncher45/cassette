package ytmusic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"cassette/core/logger"
)

const (
	innertubeAPIURL = "https://music.youtube.com/youtubei/v1"
	innertubeAPIKey = "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30"
)

// VideoType distinguishes studio audio tracks from music videos and user uploads.
type VideoType string

const (
	VideoTypeATV VideoType = "MUSIC_VIDEO_TYPE_ATV" // Audio Track Video (studio master)
	VideoTypeOMV VideoType = "MUSIC_VIDEO_TYPE_OMV" // Official Music Video
	VideoTypeUGC VideoType = "MUSIC_VIDEO_TYPE_UGC" // User Generated Content
)

// Track represents a YouTube Music track.
type Track struct {
	VideoID    string
	Title      string
	Artist     string
	Album      string
	DurationMs int
	ArtURL     string
	VideoType  VideoType
}

// Album represents a YouTube Music album.
type Album struct {
	BrowseID string
	Title    string
	Artist   string
	ArtURL   string
	Year     string
}

// Artist represents a YouTube Music artist.
type Artist struct {
	BrowseID string
	Name     string
	ArtURL   string
}

// Playlist represents a YouTube Music playlist.
type Playlist struct {
	PlaylistID string
	Title      string
	ArtURL     string
}

// SearchResults holds categorised search results.
type SearchResults struct {
	Tracks    []Track
	Albums    []Album
	Artists   []Artist
	Playlists []Playlist
}

// Client is the YouTube Music Innertube API client.
type Client struct {
	httpClient *http.Client
	mu         sync.RWMutex
}

var (
	defaultClient *Client
	once          sync.Once
)

// GetClient returns the singleton YouTube Music client.
func GetClient() *Client {
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
			DialContext:           dialer.DialContext,
			MaxIdleConns:         10,
			IdleConnTimeout:      30 * time.Second,
			TLSHandshakeTimeout:  5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		}
		defaultClient = &Client{
			httpClient: &http.Client{
				Transport: transport,
				Timeout:   15 * time.Second,
			},
		}
	})
	return defaultClient
}

// innertubeContext is the WEB_REMIX client context for YouTube Music.
type innertubeContext struct {
	Client struct {
		ClientName    string `json:"clientName"`
		ClientVersion string `json:"clientVersion"`
		HL            string `json:"hl"`
		GL            string `json:"gl"`
	} `json:"client"`
}

func newInnertubeContext() innertubeContext {
	var ctx innertubeContext
	ctx.Client.ClientName = "WEB_REMIX"
	ctx.Client.ClientVersion = "1.20240101.01.00"
	ctx.Client.HL = "en"
	ctx.Client.GL = "US"
	return ctx
}

type searchRequest struct {
	Context innertubeContext `json:"context"`
	Query   string          `json:"query"`
	Params  string          `json:"params,omitempty"`
}

type nextRequest struct {
	Context             innertubeContext `json:"context"`
	VideoID             string          `json:"videoId,omitempty"`
	PlaylistID          string          `json:"playlistId,omitempty"`
	IsAudioOnly         bool            `json:"isAudioOnly"`
	EnablePersistentPlaylistPanel bool `json:"enablePersistentPlaylistPanel"`
}

type browseRequest struct {
	Context  innertubeContext `json:"context"`
	BrowseID string          `json:"browseId"`
}

// Search performs a YouTube Music search and returns categorised results.
// It prefers Audio Track Videos (studio versions) over music videos.
func (c *Client) Search(ctx context.Context, query string, limit int) (*SearchResults, error) {
	if query == "" {
		return nil, fmt.Errorf("empty search query")
	}

	payload := searchRequest{
		Context: newInnertubeContext(),
		Query:   query,
		Params:  "EgWKAQIIAUICCAFqDBAOEAoQAxAEEAkQBQ%3D%3D", // filter for songs
	}

	body, err := c.doRequest(ctx, "/search", payload)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	results := &SearchResults{}
	tracks := parseSearchTracks(body)

	// Prefer ATV (studio) tracks, then OMV, then UGC
	var atvTracks, omvTracks, ugcTracks []Track
	for _, t := range tracks {
		switch t.VideoType {
		case VideoTypeATV:
			atvTracks = append(atvTracks, t)
		case VideoTypeOMV:
			omvTracks = append(omvTracks, t)
		default:
			ugcTracks = append(ugcTracks, t)
		}
	}

	// Merge with ATV priority
	results.Tracks = append(results.Tracks, atvTracks...)
	results.Tracks = append(results.Tracks, omvTracks...)
	results.Tracks = append(results.Tracks, ugcTracks...)

	if limit > 0 && len(results.Tracks) > limit {
		results.Tracks = results.Tracks[:limit]
	}

	logger.Log.Info().Int("tracks", len(results.Tracks)).Str("query", query).Msg("ytmusic search complete")
	return results, nil
}

// GetRadioTracks fetches YouTube Music's "up next" recommendations seeded from a video ID.
func (c *Client) GetRadioTracks(ctx context.Context, videoID string, count int) ([]Track, error) {
	if videoID == "" {
		return nil, fmt.Errorf("empty video ID for radio")
	}

	payload := nextRequest{
		Context:             newInnertubeContext(),
		VideoID:             videoID,
		IsAudioOnly:         true,
		EnablePersistentPlaylistPanel: true,
	}

	body, err := c.doRequest(ctx, "/next", payload)
	if err != nil {
		return nil, fmt.Errorf("radio request failed: %w", err)
	}

	tracks := parseRadioTracks(body)
	if count > 0 && len(tracks) > count {
		tracks = tracks[:count]
	}

	logger.Log.Info().Int("tracks", len(tracks)).Str("seed", videoID).Msg("ytmusic radio seeded")
	return tracks, nil
}

// GetAlbumTracks fetches all tracks from a YouTube Music album.
func (c *Client) GetAlbumTracks(ctx context.Context, browseID string) ([]Track, error) {
	if browseID == "" {
		return nil, fmt.Errorf("empty album browse ID")
	}

	payload := browseRequest{
		Context:  newInnertubeContext(),
		BrowseID: browseID,
	}

	body, err := c.doRequest(ctx, "/browse", payload)
	if err != nil {
		return nil, fmt.Errorf("album browse failed: %w", err)
	}

	return parseAlbumTracks(body), nil
}

// GetStreamURL returns the YouTube video URL suitable for mpv/yt-dlp playback.
func (c *Client) GetStreamURL(videoID string) string {
	return fmt.Sprintf("https://music.youtube.com/watch?v=%s", videoID)
}

// doRequest executes an Innertube API request and returns the raw JSON response.
func (c *Client) doRequest(ctx context.Context, endpoint string, payload interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s?key=%s&prettyPrint=false", innertubeAPIURL, endpoint, innertubeAPIKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://music.youtube.com/")
	req.Header.Set("Origin", "https://music.youtube.com")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("innertube API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseSearchTracks extracts tracks from the Innertube search response.
func parseSearchTracks(body map[string]interface{}) []Track {
	var tracks []Track

	contents := navigatePath(body, "contents", "tabbedSearchResultsRenderer", "tabs")
	tabs, ok := contents.([]interface{})
	if !ok || len(tabs) == 0 {
		return tracks
	}

	for _, tab := range tabs {
		tabMap, ok := tab.(map[string]interface{})
		if !ok {
			continue
		}
		sectionList := navigatePath(tabMap, "tabRenderer", "content", "sectionListRenderer", "contents")
		sections, ok := sectionList.([]interface{})
		if !ok {
			continue
		}
		for _, section := range sections {
			sectionMap, ok := section.(map[string]interface{})
			if !ok {
				continue
			}
			items := navigatePath(sectionMap, "musicShelfRenderer", "contents")
			itemList, ok := items.([]interface{})
			if !ok {
				continue
			}
			for _, item := range itemList {
				if t, ok := parseTrackFromShelf(item); ok {
					tracks = append(tracks, t)
				}
			}
		}
	}

	return tracks
}

// parseTrackFromShelf parses a single track from a musicResponsiveListItemRenderer.
func parseTrackFromShelf(item interface{}) (Track, bool) {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return Track{}, false
	}

	renderer := navigatePath(itemMap, "musicResponsiveListItemRenderer")
	rendererMap, ok := renderer.(map[string]interface{})
	if !ok {
		return Track{}, false
	}

	track := Track{}

	// Extract video ID from the overlay or playback endpoint
	overlay := navigatePath(rendererMap, "overlay", "musicItemThumbnailOverlayRenderer", "content",
		"musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
	if vid, ok := overlay.(string); ok {
		track.VideoID = vid
	}

	if track.VideoID == "" {
		// Try from flexColumns
		navEndpoint := navigatePath(rendererMap, "flexColumns")
		if cols, ok := navEndpoint.([]interface{}); ok {
			for _, col := range cols {
				colMap, _ := col.(map[string]interface{})
				runs := navigatePath(colMap, "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
				if runsList, ok := runs.([]interface{}); ok {
					for _, run := range runsList {
						runMap, _ := run.(map[string]interface{})
						vid := navigatePath(runMap, "navigationEndpoint", "watchEndpoint", "videoId")
						if vidStr, ok := vid.(string); ok && vidStr != "" {
							track.VideoID = vidStr
							break
						}
					}
				}
				if track.VideoID != "" {
					break
				}
			}
		}
	}

	if track.VideoID == "" {
		return Track{}, false
	}

	// Extract title and artist from flexColumns
	flexCols := navigatePath(rendererMap, "flexColumns")
	if cols, ok := flexCols.([]interface{}); ok {
		for i, col := range cols {
			colMap, _ := col.(map[string]interface{})
			runs := navigatePath(colMap, "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
			if runsList, ok := runs.([]interface{}); ok {
				text := extractRunsText(runsList)
				switch i {
				case 0:
					track.Title = text
				case 1:
					// Usually "Artist • Album • Duration"
					parts := strings.Split(text, " • ")
					if len(parts) > 0 {
						track.Artist = strings.TrimSpace(parts[0])
					}
					if len(parts) > 1 {
						track.Album = strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}

	// Extract thumbnail
	thumbnails := navigatePath(rendererMap, "thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails")
	if thumbList, ok := thumbnails.([]interface{}); ok && len(thumbList) > 0 {
		last := thumbList[len(thumbList)-1]
		if thumbMap, ok := last.(map[string]interface{}); ok {
			if url, ok := thumbMap["url"].(string); ok {
				track.ArtURL = url
			}
		}
	}

	// Extract video type for ATV detection
	vt := navigatePath(rendererMap, "flexColumnDisplayStyle")
	if vtStr, ok := vt.(string); ok && vtStr == "MUSIC_RESPONSIVE_LIST_ITEM_FLEX_COLUMN_DISPLAY_STYLE_STACKED" {
		track.VideoType = VideoTypeATV
	}

	// Better: check from the playbackEndpoint or badge
	badgePath := navigatePath(rendererMap, "badges")
	if badges, ok := badgePath.([]interface{}); ok {
		for _, badge := range badges {
			if bMap, ok := badge.(map[string]interface{}); ok {
				labelRuns := navigatePath(bMap, "musicInlineBadgeRenderer", "accessibilityData", "accessibilityData", "label")
				if label, ok := labelRuns.(string); ok {
					if strings.Contains(strings.ToLower(label), "video") {
						track.VideoType = VideoTypeOMV
					}
				}
			}
		}
	}

	// Default to ATV if not explicitly OMV or UGC
	if track.VideoType == "" {
		track.VideoType = VideoTypeATV
	}

	return track, true
}

// parseRadioTracks parses "up next" recommendations from the /next response.
func parseRadioTracks(body map[string]interface{}) []Track {
	var tracks []Track

	// Navigate to the playlist panel
	panels := navigatePath(body, "contents", "singleColumnMusicWatchNextResultsRenderer",
		"tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs")
	if panels == nil {
		// Try alternative path
		panels = navigatePath(body, "contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer")
	}

	tabsList, ok := panels.([]interface{})
	if !ok {
		return tracks
	}

	for _, tab := range tabsList {
		tabMap, ok := tab.(map[string]interface{})
		if !ok {
			continue
		}
		playlist := navigatePath(tabMap, "tabRenderer", "content", "musicQueueRenderer", "content",
			"playlistPanelRenderer", "contents")
		items, ok := playlist.([]interface{})
		if !ok {
			continue
		}
		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			renderer := navigatePath(itemMap, "playlistPanelVideoRenderer")
			rMap, ok := renderer.(map[string]interface{})
			if !ok {
				continue
			}

			t := Track{}
			if vid, ok := rMap["videoId"].(string); ok {
				t.VideoID = vid
			}
			if t.VideoID == "" {
				navEP := navigatePath(rMap, "navigationEndpoint", "watchEndpoint", "videoId")
				if vid, ok := navEP.(string); ok {
					t.VideoID = vid
				}
			}
			if t.VideoID == "" {
				continue
			}

			titleRuns := navigatePath(rMap, "title", "runs")
			if runs, ok := titleRuns.([]interface{}); ok {
				t.Title = extractRunsText(runs)
			}

			subtitleRuns := navigatePath(rMap, "longBylineText", "runs")
			if subtitleRuns == nil {
				subtitleRuns = navigatePath(rMap, "shortBylineText", "runs")
			}
			if runs, ok := subtitleRuns.([]interface{}); ok {
				t.Artist = extractRunsText(runs)
			}

			thumbs := navigatePath(rMap, "thumbnail", "thumbnails")
			if thumbList, ok := thumbs.([]interface{}); ok && len(thumbList) > 0 {
				last := thumbList[len(thumbList)-1]
				if tm, ok := last.(map[string]interface{}); ok {
					if url, ok := tm["url"].(string); ok {
						t.ArtURL = url
					}
				}
			}

			t.VideoType = VideoTypeATV
			tracks = append(tracks, t)
		}
	}

	return tracks
}

// parseAlbumTracks parses tracks from a /browse album response.
func parseAlbumTracks(body map[string]interface{}) []Track {
	var tracks []Track

	header := navigatePath(body, "header", "musicImmersiveHeaderRenderer")
	albumTitle := ""
	albumArtist := ""
	if hMap, ok := header.(map[string]interface{}); ok {
		titleRuns := navigatePath(hMap, "title", "runs")
		if runs, ok := titleRuns.([]interface{}); ok {
			albumTitle = extractRunsText(runs)
		}
		subtitleRuns := navigatePath(hMap, "subtitle", "runs")
		if runs, ok := subtitleRuns.([]interface{}); ok {
			albumArtist = extractRunsText(runs)
		}
	}

	sections := navigatePath(body, "contents", "singleColumnBrowseResultsRenderer", "tabs")
	tabsList, ok := sections.([]interface{})
	if !ok {
		return tracks
	}

	for _, tab := range tabsList {
		tabMap, _ := tab.(map[string]interface{})
		sectionContents := navigatePath(tabMap, "tabRenderer", "content", "sectionListRenderer", "contents")
		secs, ok := sectionContents.([]interface{})
		if !ok {
			continue
		}
		for _, sec := range secs {
			secMap, _ := sec.(map[string]interface{})
			items := navigatePath(secMap, "musicShelfRenderer", "contents")
			itemList, ok := items.([]interface{})
			if !ok {
				continue
			}
			for _, item := range itemList {
				if t, ok := parseTrackFromShelf(item); ok {
					if t.Album == "" {
						t.Album = albumTitle
					}
					if t.Artist == "" {
						t.Artist = albumArtist
					}
					tracks = append(tracks, t)
				}
			}
		}
	}

	return tracks
}

// navigatePath traverses a nested map[string]interface{} using dot-like path segments.
func navigatePath(data interface{}, keys ...string) interface{} {
	current := data
	for _, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[key]
		if !ok {
			return nil
		}
	}
	return current
}

// extractRunsText concatenates "text" fields from a "runs" array.
func extractRunsText(runs []interface{}) string {
	var parts []string
	for _, r := range runs {
		rMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := rMap["text"].(string); ok {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}
