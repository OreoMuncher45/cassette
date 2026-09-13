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

func NewAppKeyMap() AppKeyMap {
	m := AppKeyMap{
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
	newKey = strings.TrimSpace(newKey)
	if newKey == "" {
		return
	}
	switch id {
	case "keybinds":
		k.OpenKeybinds.SetKeys(newKey)
	case "settings":
		k.OpenSettings.SetKeys(newKey)
	case "play_pause":
		k.PlayPause.SetKeys(newKey)
	case "shuffle":
		k.Shuffle.SetKeys(newKey)
	case "next_track":
		k.NextTrack.SetKeys(newKey)
	case "prev_track":
		k.PrevTrack.SetKeys(newKey)
	case "volume_up":
		k.VolumeUp.SetKeys(newKey)
	case "volume_down":
		k.VolumeDown.SetKeys(newKey)
	case "seek_forward":
		k.SeekForward.SetKeys(newKey)
	case "seek_backward":
		k.SeekBackward.SetKeys(newKey)
	case "lyrics":
		k.ToggleLyrics.SetKeys(newKey)
	case "art":
		k.ToggleArt.SetKeys(newKey)
	case "queue":
		k.ToggleQueue.SetKeys(newKey)
	case "tab":
		k.TogglePanel.SetKeys(newKey)
	case "devices":
		k.Devices.SetKeys(newKey)
	case "zen":
		k.ZenMode.SetKeys(newKey)
	case "search":
		k.Search.SetKeys(newKey)
	case "quit":
		k.Quit.SetKeys(newKey)
	}
	_ = k.SaveCustomKeys()
}

func (k *AppKeyMap) GetKeybindsPath() string {
	return filepath.Join(utils.SafeGetConfigDir(), "keybinds.json")
}

func (k *AppKeyMap) SaveCustomKeys() error {
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
		case "settings":
			k.OpenSettings.SetKeys(keys...)
		case "play_pause":
			k.PlayPause.SetKeys(keys...)
		case "shuffle":
			k.Shuffle.SetKeys(keys...)
		case "next_track":
			k.NextTrack.SetKeys(keys...)
		case "prev_track":
			k.PrevTrack.SetKeys(keys...)
		case "volume_up":
			k.VolumeUp.SetKeys(keys...)
		case "volume_down":
			k.VolumeDown.SetKeys(keys...)
		case "seek_forward":
			k.SeekForward.SetKeys(keys...)
		case "seek_backward":
			k.SeekBackward.SetKeys(keys...)
		case "lyrics":
			k.ToggleLyrics.SetKeys(keys...)
		case "art":
			k.ToggleArt.SetKeys(keys...)
		case "queue":
			k.ToggleQueue.SetKeys(keys...)
		case "tab":
			k.TogglePanel.SetKeys(keys...)
		case "devices":
			k.Devices.SetKeys(keys...)
		case "zen":
			k.ZenMode.SetKeys(keys...)
		case "search":
			k.Search.SetKeys(keys...)
		case "quit":
			k.Quit.SetKeys(keys...)
		}
	}
}

func (k *AppKeyMap) ResetDefaults() {
	_ = os.Remove(k.GetKeybindsPath())
	fresh := NewAppKeyMap()
	*k = fresh
}

func (k AppKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.OpenKeybinds, k.OpenSettings, k.PlayPause, k.Shuffle, k.Quit}
}

func (k AppKeyMap) FullHelp() [][]key.Binding {
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

func (k AppKeyMap) WithMediaPanelOpen(open bool) AppKeyMap {
	k.MediaPanelOpen = open
	return k
}

func (k AppKeyMap) WithInfoOpen(open bool) AppKeyMap {
	k.InfoOpen = open
	return k
}
