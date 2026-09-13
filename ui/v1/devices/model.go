package devices

import (
	tea "charm.land/bubbletea/v2"
	"github.com/zmb3/spotify/v2"
)

type State int

const (
	StateLoading State = iota
	StateSelecting
	StateNoDevices
)

type DevicesLoadedMsg struct {
	Devices []spotify.PlayerDevice
	Err     error
}

type DeviceSelectedMsg struct {
	Device spotify.PlayerDevice
}

type DeviceDismissedMsg struct{}

type Model struct {
	devices        []spotify.PlayerDevice
	cursor         int
	selectedDevice *spotify.PlayerDevice
	state          State
	width          int
	height         int
	statusMessage  string
	canDismiss     bool
}

func NewModel(canDismiss bool) *Model {
	return &Model{
		devices:    nil,
		cursor:     0,
		state:      StateLoading,
		canDismiss: canDismiss,
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetDevices(devices []spotify.PlayerDevice) {
	m.devices = devices
	if len(devices) == 0 {
		m.state = StateNoDevices
		m.cursor = 0
		return
	}
	m.state = StateSelecting
	// If one is active, default cursor to it
	for i, d := range devices {
		if d.Active {
			m.cursor = i
			break
		}
	}
	if m.cursor >= len(devices) {
		m.cursor = 0
	}
}

func (m *Model) SetLoading() {
	m.state = StateLoading
	m.statusMessage = ""
}

func (m *Model) SetStatus(msg string) {
	m.statusMessage = msg
}

func (m *Model) State() State {
	return m.state
}

func (m *Model) SelectedDevice() *spotify.PlayerDevice {
	return m.selectedDevice
}

func (m *Model) SetCanDismiss(canDismiss bool) {
	m.canDismiss = canDismiss
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DevicesLoadedMsg:
		if msg.Err != nil {
			m.statusMessage = msg.Err.Error()
			m.state = StateNoDevices
			return m, nil
		}
		m.SetDevices(msg.Devices)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if len(m.devices) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.devices) - 1
				}
			}
		case "down", "j":
			if len(m.devices) > 0 {
				m.cursor++
				if m.cursor >= len(m.devices) {
					m.cursor = 0
				}
			}
		case "enter":
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				selected := m.devices[m.cursor]
				m.selectedDevice = &selected
				return m, func() tea.Msg {
					return DeviceSelectedMsg{Device: selected}
				}
			}
		case "esc", "d", "q":
			if m.canDismiss {
				return m, func() tea.Msg {
					return DeviceDismissedMsg{}
				}
			}
		}
	}
	return m, nil
}
