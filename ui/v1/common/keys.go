package common

import (
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
	MediaPanelOpen bool
	InfoOpen       bool
}

func NewAppKeyMap() AppKeyMap {
	return AppKeyMap{
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
			key.WithKeys("s"),
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
	}
}

func (k AppKeyMap) ShortHelp() []key.Binding {
	if k.MediaPanelOpen {
		return []key.Binding{k.TogglePanel, k.Select, k.Search, k.Back, k.PlayPause, k.ToggleLyrics, k.ToggleArt, k.ToggleQueue, k.Devices, k.Quit}
	}
	return []key.Binding{k.PlayPause, k.NextTrack, k.PrevTrack, k.VolumeUp, k.VolumeDown, k.ToggleLyrics, k.ToggleArt, k.ToggleQueue, k.TogglePanel, k.Devices, k.Quit}
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
