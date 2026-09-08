package app

import (
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	"testing"
)

func TestDaemonPaginationHandlesFilteredEntities(t *testing.T) {
	if page := paginationFromOffset(20, 3, 25, 10); page.HasNext {
		t.Fatal("filtered last page should not request another page")
	}
	if page := paginationFromCursor(1, 0, 20, 10, "10"); !page.HasNext {
		t.Fatal("a cursor must remain usable when all entities on a page are unavailable")
	}
}

func TestLibraryLoadsWhenDaemonBecomesReady(t *testing.T) {
	model := NewModel()
	cmd, handled := model.handleSystemMessages(playerReadyMsg{})
	if !handled || cmd == nil || !model.playerReady {
		t.Fatal("daemon ready did not enable browsing")
	}
	request, ok := cmd().(common.MediaRequest)
	if !ok || request.Kind != common.GetUserPlaylists {
		t.Fatalf("initial request = %#v", request)
	}
}
