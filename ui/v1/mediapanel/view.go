package mediapanel

import (
	"strings"

	"charm.land/lipgloss/v2"
	"cassette/ui/v1/common"
)

const preferredInfoHeight = 7

func (m *Model) View() string {
	panelShell := m.styles.panel.Width(m.width).Height(m.height).Render("")
	panelNav := m.renderPanelNav()
	searchLine := m.renderSearchLine()
	searchHeight := lipgloss.Height(searchLine)
	listOffsetY := lipgloss.Height(panelNav)
	reservedSearchHeight := 0
	if m.infoOpen {
		reservedSearchHeight = searchHeight
	}
	infoHeight := m.infoStripHeight(listOffsetY, reservedSearchHeight)
	listHeight := max(1, m.height-listOffsetY-reservedSearchHeight-infoHeight)
	m.activePanel().SetSize(m.width, listHeight)
	listView := m.activePanel().View()
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(panelShell).ID("panel"),
		lipgloss.NewLayer(panelNav).X(m.width/2 - lipgloss.Width(panelNav)/2).Y(0).ID("panelNav"),
		lipgloss.NewLayer(listView).X(1).Y(listOffsetY).ID("list"),
	}
	if infoHeight > 0 {
		layers = append(layers, lipgloss.NewLayer(m.renderInfoStrip(infoHeight)).X(1).Y(listOffsetY+listHeight).ID("info"))
	}
	if searchLine != "" {
		searchY := max(listOffsetY, listOffsetY+listHeight-searchHeight)
		if m.infoOpen {
			searchY = listOffsetY + listHeight + infoHeight
		}
		layers = append(layers, lipgloss.NewLayer(searchLine).X(2).Y(searchY).ID("search"))
	}
	return lipgloss.NewCompositor(layers...).Render()
}

func (m *Model) renderPanelNav() string {
	segments := []struct {
		label string
		kind  common.ListKind
	}{
		{label: "PL", kind: common.Playlists},
		{label: "TR", kind: common.Tracks},
		{label: "AL", kind: common.Albums},
		{label: "AR", kind: common.Artists},
	}

	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if m.activePanel().kind == segment.kind {
			parts = append(parts, m.styles.panelNavActive.Render(segment.label))
			continue
		}
		parts = append(parts, m.styles.panelNavMuted.Render(segment.label))
	}
	return m.styles.panelNav.Render(strings.Join(parts, " - "))
}

func (m *Model) renderSearchLine() string {
	if !m.searchFocused && m.searchQuery == "" {
		return ""
	}
	if m.searchFocused {
		return m.styles.searchLine.Width(max(0, m.width-4)).Render(m.searchInput.View())
	}
	return m.styles.searchLine.Width(max(0, m.width-4)).Render(m.styles.searchValue.Render("Search: " + m.searchQuery))
}

func (m *Model) renderInfoStrip(height int) string {
	stripWidth := max(0, m.width-2)
	viewportWidth := max(0, stripWidth-m.styles.infoStrip.GetHorizontalFrameSize())
	viewportHeight := max(0, height-m.styles.infoStrip.GetVerticalFrameSize())
	m.infoViewport.SetWidth(viewportWidth)
	m.infoViewport.SetHeight(viewportHeight)
	return m.styles.infoStrip.Width(stripWidth).Height(height).Render(m.infoViewport.View())
}

func (m *Model) infoStripHeight(listOffsetY, searchHeight int) int {
	if !m.infoOpen {
		return 0
	}
	available := max(0, m.height-listOffsetY-searchHeight)
	maxInfoHeight := max(0, available-1)
	return min(preferredInfoHeight, maxInfoHeight)
}

func (m *Model) syncInfoViewportLayout() {
	searchHeight := lipgloss.Height(m.renderSearchLine())
	listOffsetY := lipgloss.Height(m.renderPanelNav())
	infoHeight := m.infoStripHeight(listOffsetY, searchHeight)
	stripWidth := max(0, m.width-2)
	viewportWidth := max(0, stripWidth-m.styles.infoStrip.GetHorizontalFrameSize())
	viewportHeight := max(0, infoHeight-m.styles.infoStrip.GetVerticalFrameSize())
	m.infoViewport.SetWidth(viewportWidth)
	m.infoViewport.SetHeight(viewportHeight)
}

func (p *panel) View() string {
	p.activeList().SetSize(p.width, p.height)
	return p.activeList().View()
}

func (p *panel) SetSize(width, height int) {
	p.width = width
	p.height = height
}
