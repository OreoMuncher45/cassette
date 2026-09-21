package ytmusic

import (
	"context"
	"testing"
)

func TestNavigatePath(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep_value",
			},
		},
	}

	result := navigatePath(data, "a", "b", "c")
	if result != "deep_value" {
		t.Errorf("expected 'deep_value', got %v", result)
	}

	nilResult := navigatePath(data, "a", "z")
	if nilResult != nil {
		t.Errorf("expected nil for missing key, got %v", nilResult)
	}
}

func TestExtractRunsText(t *testing.T) {
	runs := []interface{}{
		map[string]interface{}{"text": "Hello"},
		map[string]interface{}{"text": " "},
		map[string]interface{}{"text": "World"},
	}

	result := extractRunsText(runs)
	if result != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", result)
	}
}

func TestParseTrackFromShelf_EmptyInput(t *testing.T) {
	_, ok := parseTrackFromShelf(nil)
	if ok {
		t.Error("expected false for nil input")
	}

	_, ok = parseTrackFromShelf(map[string]interface{}{})
	if ok {
		t.Error("expected false for empty map")
	}
}

func TestVideoTypeConstants(t *testing.T) {
	if VideoTypeATV != "MUSIC_VIDEO_TYPE_ATV" {
		t.Errorf("unexpected ATV value: %s", VideoTypeATV)
	}
	if VideoTypeOMV != "MUSIC_VIDEO_TYPE_OMV" {
		t.Errorf("unexpected OMV value: %s", VideoTypeOMV)
	}
	if VideoTypeUGC != "MUSIC_VIDEO_TYPE_UGC" {
		t.Errorf("unexpected UGC value: %s", VideoTypeUGC)
	}
}

func TestGetStreamURL(t *testing.T) {
	c := &Client{}
	url := c.GetStreamURL("dQw4w9WgXcQ")
	expected := "https://music.youtube.com/watch?v=dQw4w9WgXcQ"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestParseTrackFromShelf_ATVExtraction(t *testing.T) {
	// Item with watchEndpointMusicSupportedConfigs indicating ATV
	item := map[string]interface{}{
		"musicResponsiveListItemRenderer": map[string]interface{}{
			"overlay": map[string]interface{}{
				"musicItemThumbnailOverlayRenderer": map[string]interface{}{
					"content": map[string]interface{}{
						"musicPlayButtonRenderer": map[string]interface{}{
							"playNavigationEndpoint": map[string]interface{}{
								"watchEndpoint": map[string]interface{}{
									"videoId": "abc123xyz",
									"watchEndpointMusicSupportedConfigs": map[string]interface{}{
										"watchEndpointMusicConfig": map[string]interface{}{
											"musicVideoType": "MUSIC_VIDEO_TYPE_ATV",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	track, ok := parseTrackFromShelf(item)
	if !ok {
		t.Fatal("expected parseTrackFromShelf to succeed")
	}
	if track.VideoID != "abc123xyz" {
		t.Errorf("expected videoId 'abc123xyz', got %q", track.VideoID)
	}
	if track.VideoType != VideoTypeATV {
		t.Errorf("expected VideoTypeATV, got %v", track.VideoType)
	}
}

func TestLiveRadio(t *testing.T) {
	c := GetClient()
	results, err := c.Search(context.Background(), "Daft Punk Get Lucky", 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results.Tracks) == 0 {
		t.Fatal("no tracks returned from search")
	}

	vid := results.Tracks[0].VideoID
	t.Logf("Testing GetRadioTracks with vid: %s", vid)
	radio, err := c.GetRadioTracks(context.Background(), vid, 25)
	if err != nil {
		t.Fatalf("GetRadioTracks failed: %v", err)
	}
	t.Logf("Radio returned %d tracks", len(radio))
	if len(radio) < 5 {
		t.Fatalf("expected at least 5 radio tracks, got %d", len(radio))
	}
	for i := 0; i < len(radio) && i < 10; i++ {
		tr := radio[i]
		t.Logf("  Radio Track %d: ID=%q Title=%q Artist=%q Album=%q", i, tr.VideoID, tr.Title, tr.Artist, tr.Album)
		if tr.VideoID == "" {
			t.Errorf("track %d has empty VideoID", i)
		}
		if tr.Title == "" {
			t.Errorf("track %d has empty Title", i)
		}
	}
}
