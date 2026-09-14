package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"cassette/core/utils"
	"charm.land/bubbles/v2/key"
)

type AppKeyMap struct {
	ToggleHelp     key.Binding
	Quit           key.Binding
	CycleLibrary   key.Binding
	Search         key.Binding
	Cancel         key.Binding
	MoreInfo       key.Binding
	InfoScrollUp   key.Binding
	InfoScrollDown key.Binding
	Select         key.Binding
	Back           key.Binding
	NextPage       key.Binding
	PrevPage       key.Binding
	TogglePanel    key.Binding
	PlayPause      key.Binding
	SeekForward    key.Binding
	SeekBackward   key.Binding
	NextTrack      key.Binding
	PrevTrack      key.Binding
	VolumeDown     key.Binding
	VolumeUp       key.Binding
	Shuffle        key.Binding
	ZenMode        key.Binding
	Devices        key.Binding
	ToggleQueue    key.Binding
	ToggleLyrics   key.Binding
	ToggleArt      key.Binding
	OpenKeybinds   key.Binding
	OpenSettings   key.Binding
	MediaPanelOpen bool
	InfoOpen       bool
}

func NewAppKeyMap() *AppKeyMap {
	m := &AppKeyMap{
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		CycleLibrary: key.NewBinding(
			key.WithKeys("c", "C"),
			key.WithHelp("c", "next section"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		MoreInfo: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "more info"),
		),
		InfoScrollUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "scroll info up"),
		),
		InfoScrollDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "scroll info down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select item"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "delete"),
			key.WithHelp("del", "back"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "l", "]"),
			key.WithHelp("right/l/]", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left", "h", "["),
			key.WithHelp("left/h/[", "prev page"),
		),
		TogglePanel: key.NewBinding(
			key.WithKeys("tab", "P"),
			key.WithHelp("tab", "library/lyrics"),
		),
		PlayPause: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "play/pause"),
		),
		SeekForward: key.NewBinding(
			key.WithKeys(">", ".", "right", "]"),
			key.WithHelp(">", "seek +"),
		),
		SeekBackward: key.NewBinding(
			key.WithKeys("<", ",", "left", "h", "["),
			key.WithHelp("<", "seek -"),
		),
		NextTrack: key.NewBinding(
			key.WithKeys("n", "ctrl+s"),
			key.WithHelp("n", "next"),
		),
		PrevTrack: key.NewBinding(
			key.WithKeys("p", "N", "ctrl+r"),
			key.WithHelp("p", "prev"),
		),
		VolumeDown: key.NewBinding(
			key.WithKeys("-", "_", "j", "ctrl+p"),
			key.WithHelp("-", "vol -"),
		),
		VolumeUp: key.NewBinding(
			key.WithKeys("+", "=", "k", "ctrl+n"),
			key.WithHelp("+", "vol +"),
		),
		Shuffle: key.NewBinding(
			key.WithKeys("s", "S"),
			key.WithHelp("s", "shuffle"),
		),
		ZenMode: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "zen mode"),
		),
		Devices: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "devices"),
		),
		ToggleQueue: key.NewBinding(
			key.WithKeys("q", "Q"),
			key.WithHelp("q", "queue"),
		),
		ToggleLyrics: key.NewBinding(
			key.WithKeys("l", "L"),
			key.WithHelp("l", "lyrics"),
		),
		ToggleArt: key.NewBinding(
			key.WithKeys("a", "A"),
			key.WithHelp("a", "art"),
		),
		OpenKeybinds: key.NewBinding(
			key.WithKeys("f1", "?"),
			key.WithHelp("f1", "keybinds"),
		),
		OpenSettings: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("f2", "settings"),
		),
	}
	m.LoadCustomKeys()
	return m
}

type RebindAction struct {
	ID          string
	Name        string
	Keys        []string
	Description string
}

func (k *AppKeyMap) GetRebindActions() []RebindAction {
	return []RebindAction{
		{ID: "keybinds", Name: "Keybinds Manager", Keys: k.OpenKeybinds.Keys(), Description: "Open searchable keybindings manager & rebind keys"},
		{ID: "settings", Name: "Settings & Visuals", Keys: k.OpenSettings.Keys(), Description: "Open themes, dynamic album art colors & RGB effects"},
		{ID: "play_pause", Name: "Play / Pause", Keys: k.PlayPause.Keys(), Description: "Toggle playback on active device"},
		{ID: "shuffle", Name: "Toggle Shuffle", Keys: k.Shuffle.Keys(), Description: "Turn shuffle on or off (s)"},
		{ID: "next_track", Name: "Next Track", Keys: k.NextTrack.Keys(), Description: "Skip to next track in queue"},
		{ID: "prev_track", Name: "Previous Track", Keys: k.PrevTrack.Keys(), Description: "Restart song or go to previous track"},
		{ID: "volume_up", Name: "Volume Up", Keys: k.VolumeUp.Keys(), Description: "Increase volume by 5%"},
		{ID: "volume_down", Name: "Volume Down", Keys: k.VolumeDown.Keys(), Description: "Decrease volume by 5%"},
		{ID: "seek_forward", Name: "Seek Forward", Keys: k.SeekForward.Keys(), Description: "Fast forward 5 seconds"},
		{ID: "seek_backward", Name: "Seek Backward", Keys: k.SeekBackward.Keys(), Description: "Rewind 5 seconds"},
		{ID: "lyrics", Name: "Toggle Lyrics", Keys: k.ToggleLyrics.Keys(), Description: "Open / close synced lyrics on left side"},
		{ID: "art", Name: "Toggle Big Artwork", Keys: k.ToggleArt.Keys(), Description: "Open / close large album cover on left side"},
		{ID: "queue", Name: "Toggle Queue", Keys: k.ToggleQueue.Keys(), Description: "Open / close queue panel on right side"},
		{ID: "tab", Name: "Cycle Tabs / Swap Panel", Keys: k.TogglePanel.Keys(), Description: "Cycle search tabs (PL/TR/AL/AR) or open library"},
		{ID: "devices", Name: "Switch Device", Keys: k.Devices.Keys(), Description: "Select Spotify Connect output device"},
		{ID: "zen", Name: "Zen Mode", Keys: k.ZenMode.Keys(), Description: "Focus mode: hide panels, show only cassette"},
		{ID: "search", Name: "Search Library", Keys: k.Search.Keys(), Description: "Search tracks, artists, and playlists"},
		{ID: "quit", Name: "Quit Cassette", Keys: k.Quit.Keys(), Description: "Exit application cleanly"},
	}
}

func (k *AppKeyMap) SetActionKey(id string, newKey string) {
	if k == nil {
		return
	}
	newKey = strings.TrimSpace(newKey)
	if newKey == "" {
		return
	}
	switch id {
	case "keybinds":
		k.OpenKeybinds.SetKeys(newKey)
		k.OpenKeybinds.SetHelp(newKey, "keybinds")
	case "settings":
		k.OpenSettings.SetKeys(newKey)
		k.OpenSettings.SetHelp(newKey, "settings")
	case "play_pause":
		k.PlayPause.SetKeys(newKey)
		k.PlayPause.SetHelp(newKey, "play/pause")
	case "shuffle":
		k.Shuffle.SetKeys(newKey)
		k.Shuffle.SetHelp(newKey, "shuffle")
	case "next_track":
		k.NextTrack.SetKeys(newKey)
		k.NextTrack.SetHelp(newKey, "next")
	case "prev_track":
		k.PrevTrack.SetKeys(newKey)
		k.PrevTrack.SetHelp(newKey, "prev")
	case "volume_up":
		k.VolumeUp.SetKeys(newKey)
		k.VolumeUp.SetHelp(newKey, "vol +")
	case "volume_down":
		k.VolumeDown.SetKeys(newKey)
		k.VolumeDown.SetHelp(newKey, "vol -")
	case "seek_forward":
		k.SeekForward.SetKeys(newKey)
		k.SeekForward.SetHelp(newKey, "seek +")
	case "seek_backward":
		k.SeekBackward.SetKeys(newKey)
		k.SeekBackward.SetHelp(newKey, "seek -")
	case "lyrics":
		k.ToggleLyrics.SetKeys(newKey)
		k.ToggleLyrics.SetHelp(newKey, "lyrics")
	case "art":
		k.ToggleArt.SetKeys(newKey)
		k.ToggleArt.SetHelp(newKey, "art")
	case "queue":
		k.ToggleQueue.SetKeys(newKey)
		k.ToggleQueue.SetHelp(newKey, "queue")
	case "tab":
		k.TogglePanel.SetKeys(newKey)
		k.TogglePanel.SetHelp(newKey, "library/lyrics")
	case "devices":
		k.Devices.SetKeys(newKey)
		k.Devices.SetHelp(newKey, "devices")
	case "zen":
		k.ZenMode.SetKeys(newKey)
		k.ZenMode.SetHelp(newKey, "zen mode")
	case "search":
		k.Search.SetKeys(newKey)
		k.Search.SetHelp(newKey, "search")
	case "quit":
		k.Quit.SetKeys(newKey)
		k.Quit.SetHelp(newKey, "quit")
	}
	_ = k.SaveCustomKeys()
}

func (k *AppKeyMap) GetKeybindsPath() string {
	return filepath.Join(utils.SafeGetConfigDir(), "keybinds.json")
}

func (k *AppKeyMap) SaveCustomKeys() error {
	if k == nil {
		return nil
	}
	m := make(map[string][]string)
	for _, a := range k.GetRebindActions() {
		m[a.ID] = a.Keys
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(k.GetKeybindsPath(), data, 0644)
}

func (k *AppKeyMap) LoadCustomKeys() {
	if k == nil {
		return
	}
	data, err := os.ReadFile(k.GetKeybindsPath())
	if err != nil {
		return
	}
	var m map[string][]string
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	for id, keys := range m {
		if len(keys) == 0 {
			continue
		}
		switch id {
		case "keybinds":
			k.OpenKeybinds.SetKeys(keys...)
			k.OpenKeybinds.SetHelp(keys[0], "keybinds")
		case "settings":
			k.OpenSettings.SetKeys(keys...)
			k.OpenSettings.SetHelp(keys[0], "settings")
		case "play_pause":
			k.PlayPause.SetKeys(keys...)
			k.PlayPause.SetHelp(keys[0], "play/pause")
		case "shuffle":
			k.Shuffle.SetKeys(keys...)
			k.Shuffle.SetHelp(keys[0], "shuffle")
		case "next_track":
			k.NextTrack.SetKeys(keys...)
			k.NextTrack.SetHelp(keys[0], "next")
		case "prev_track":
			k.PrevTrack.SetKeys(keys...)
			k.PrevTrack.SetHelp(keys[0], "prev")
		case "volume_up":
			k.VolumeUp.SetKeys(keys...)
			k.VolumeUp.SetHelp(keys[0], "vol +")
		case "volume_down":
			k.VolumeDown.SetKeys(keys...)
			k.VolumeDown.SetHelp(keys[0], "vol -")
		case "seek_forward":
			k.SeekForward.SetKeys(keys...)
			k.SeekForward.SetHelp(keys[0], "seek +")
		case "seek_backward":
			k.SeekBackward.SetKeys(keys...)
			k.SeekBackward.SetHelp(keys[0], "seek -")
		case "lyrics":
			k.ToggleLyrics.SetKeys(keys...)
			k.ToggleLyrics.SetHelp(keys[0], "lyrics")
		case "art":
			k.ToggleArt.SetKeys(keys...)
			k.ToggleArt.SetHelp(keys[0], "art")
		case "queue":
			k.ToggleQueue.SetKeys(keys...)
			k.ToggleQueue.SetHelp(keys[0], "queue")
		case "tab":
			k.TogglePanel.SetKeys(keys...)
			k.TogglePanel.SetHelp(keys[0], "library/lyrics")
		case "devices":
			k.Devices.SetKeys(keys...)
			k.Devices.SetHelp(keys[0], "devices")
		case "zen":
			k.ZenMode.SetKeys(keys...)
			k.ZenMode.SetHelp(keys[0], "zen mode")
		case "search":
			k.Search.SetKeys(keys...)
			k.Search.SetHelp(keys[0], "search")
		case "quit":
			k.Quit.SetKeys(keys...)
			k.Quit.SetHelp(keys[0], "quit")
		}
	}
}

func (k *AppKeyMap) ResetDefaults() {
	_ = os.Remove(k.GetKeybindsPath())
	fresh := NewAppKeyMap()
	*k = *fresh
}

func (k *AppKeyMap) ShortHelp() []key.Binding {
	if k == nil {
		return nil
	}
	return []key.Binding{k.OpenKeybinds, k.OpenSettings, k.PlayPause, k.Shuffle, k.Quit}
}

func (k *AppKeyMap) FullHelp() [][]key.Binding {
	if k == nil {
		return nil
	}
	help := [][]key.Binding{
		{k.ToggleHelp, k.Quit, k.TogglePanel, k.Devices},
	}
	if k.MediaPanelOpen {
		if k.InfoOpen {
			closeInfo := key.NewBinding(
				key.WithKeys("i", "esc", "backspace", "delete"),
				key.WithHelp("i/esc/del", "close info"),
			)
			scrollInfo := key.NewBinding(
				key.WithKeys("ctrl+u", "ctrl+d"),
				key.WithHelp("ctrl+u/d", "scroll info"),
			)
			return append(help,
				[]key.Binding{k.CycleLibrary, k.Search, k.Select, closeInfo},
				[]key.Binding{k.NextPage, k.PrevPage, scrollInfo},
			)
		}
		return append(help, []key.Binding{k.CycleLibrary, k.Search, k.MoreInfo, k.Select, k.Back, k.NextPage, k.PrevPage})
	}
	return append(help,
		[]key.Binding{k.PlayPause, k.SeekForward, k.SeekBackward},
		[]key.Binding{k.VolumeDown, k.VolumeUp, k.NextTrack, k.PrevTrack, k.Shuffle, k.Devices},
	)
}

func (k *AppKeyMap) WithMediaPanelOpen(open bool) *AppKeyMap {
	if k != nil {
		k.MediaPanelOpen = open
	}
	return k
}

func (k *AppKeyMap) WithInfoOpen(open bool) *AppKeyMap {
	if k != nil {
		k.InfoOpen = open
	}
	return k
}
