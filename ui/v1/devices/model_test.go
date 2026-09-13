package devices

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/zmb3/spotify/v2"
)

func TestDeviceSelectionNavigation(t *testing.T) {
	m := NewModel(false)
	devs := []spotify.PlayerDevice{
		{ID: "d1", Name: "Phone", Active: false},
		{ID: "d2", Name: "Computer", Active: true},
	}
	m.SetDevices(devs)

	if m.cursor != 1 {
		t.Fatalf("expected cursor to start at active device 1, got %d", m.cursor)
	}

	m.Update(tea.KeyPressMsg{Code: 'k'})
	if m.cursor != 0 {
		t.Fatalf("expected cursor to move up to 0, got %d", m.cursor)
	}

	m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.cursor != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", m.cursor)
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter")
	}
	msg := cmd()
	selectedMsg, ok := msg.(DeviceSelectedMsg)
	if !ok {
		t.Fatalf("expected DeviceSelectedMsg, got %T", msg)
	}
	if selectedMsg.Device.ID != "d2" {
		t.Fatalf("selected device = %q, want d2", selectedMsg.Device.ID)
	}
}

func TestDeviceSelectionEmptyState(t *testing.T) {
	m := NewModel(false)
	m.SetDevices(nil)
	if m.State() != StateNoDevices {
		t.Fatalf("state = %v, want StateNoDevices", m.State())
	}
}
