package app

import (
	"fmt"

	"cassette/core/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) View() tea.View {
	if m.fatalErr != nil {
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightRed).Render("Error")
		message := lipgloss.NewStyle().MarginTop(1).Align(lipgloss.Center).Render(fmt.Sprintf("%v", m.fatalErr))
		hint := lipgloss.NewStyle().MarginTop(1).Foreground(lipgloss.BrightBlack).Render("Exiting...")
		content := lipgloss.JoinVertical(lipgloss.Center, title, message, hint)
		view := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(content)
		return tea.NewView(view)
	}
	if m.authModel != nil && m.authModel.State() < 2 {
		return m.authModel.View()
	}
	if m.devicePickerOpen && m.devicesModel != nil {
		return m.devicesModel.View()
	}

	// When terminal is unfocused/backgrounded, render a lightweight placeholder
	// to avoid wasting CPU on ANSI cassette animation and album art rasterization
	if !m.isFocused {
		bgPlaceholder := m.backgroundPlaceholderView()
		return tea.NewView(bgPlaceholder)
	}

	mediaCenterView := m.mediaCenter.View(m.width, m.height)

	th := theme.Get()
	cKey := lipgloss.NewStyle().Foreground(th.PrimaryColor()).Bold(true)
	cLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	cDot := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	bottomContent := fmt.Sprintf("%s %s   %s   %s %s",
		cKey.Render("F1"),
		cLabel.Render("Keybinds"),
		cDot.Render("•"),
		cKey.Render("F2"),
		cLabel.Render("Settings"),
	)
	bottomBar := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(bottomContent)

	if m.viewportTooSmall(mediaCenterView, bottomBar) {
		return tea.NewView(m.smallViewportView(mediaCenterView, bottomBar))
	}
	modelView := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(mediaCenterView)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(modelView).ID("model"),
	}
	if !m.mediaCenter.IsZenMode() {
		layers = append(layers, lipgloss.NewLayer(bottomBar).Y(m.height-lipgloss.Height(bottomBar)).ID("bottomBar"))
	}
	if m.keybindsModel.IsOpen() {
		kbView := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.keybindsModel.View())
		layers = append(layers, lipgloss.NewLayer(kbView).ID("keybinds"))
	}
	if m.settingsModel.IsOpen() {
		settView := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.settingsModel.View())
		layers = append(layers, lipgloss.NewLayer(settView).ID("settings"))
	}
	// Welcome popup overlay (one-time changelog + donation)
	if m.welcomeModel != nil && m.welcomeModel.IsOpen() {
		welcomeView := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.welcomeModel.View())
		layers = append(layers, lipgloss.NewLayer(welcomeView).ID("welcome"))
	}
	return tea.NewView(lipgloss.NewCompositor(layers...).Render())
}

func (m *Model) viewportTooSmall(mediaCenterView, helpLine string) bool {
	if m.width <= 0 || m.height <= 0 {
		return false
	}
	requiredWidth := lipgloss.Width(mediaCenterView)
	requiredHeight := lipgloss.Height(mediaCenterView) + lipgloss.Height(helpLine)
	return m.width < requiredWidth || m.height < requiredHeight
}

func (m *Model) smallViewportView(mediaCenterView, helpLine string) string {
	requiredWidth := lipgloss.Width(mediaCenterView)
	requiredHeight := lipgloss.Height(mediaCenterView) + lipgloss.Height(helpLine)
	message := lipgloss.JoinVertical(
		lipgloss.Center,
		"terminal size too small",
		lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render(
			fmt.Sprintf("need at least %dx%d", requiredWidth, requiredHeight),
		),
	)
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(message)
}
