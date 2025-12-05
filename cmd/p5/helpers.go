package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/rfhold/p5/internal/ui"
)

// Focus management helpers

// showErrorModal shows the error modal and pushes focus to it
func (m *Model) showErrorModal(title, summary, details string) {
	m.ui.ErrorModal.Show(title, summary, details)
	m.ui.Focus.Push(ui.FocusErrorModal)
}

// hideErrorModal hides the error modal and pops focus
func (m *Model) hideErrorModal() {
	m.ui.ErrorModal.Hide()
	m.ui.Focus.Remove(ui.FocusErrorModal)
}

// showConfirmModal shows the confirm modal and pushes focus to it
func (m *Model) showConfirmModal() {
	m.ui.Focus.Push(ui.FocusConfirmModal)
}

// hideConfirmModal hides the confirm modal and pops focus
func (m *Model) hideConfirmModal() {
	m.ui.ConfirmModal.Hide()
	m.ui.Focus.Remove(ui.FocusConfirmModal)
}

// showImportModal shows the import modal and pushes focus to it
func (m *Model) showImportModal(resourceType, name, urn, parent string) {
	m.ui.ImportModal.Show(resourceType, name, urn, parent)
	m.ui.Focus.Push(ui.FocusImportModal)
}

// hideImportModal hides the import modal and pops focus
func (m *Model) hideImportModal() {
	m.ui.ImportModal.Hide()
	m.ui.Focus.Remove(ui.FocusImportModal)
}

// showStackInitModal shows the stack init modal and pushes focus to it
func (m *Model) showStackInitModal() {
	m.ui.StackInitModal.Show()
	m.ui.Focus.Push(ui.FocusStackInitModal)
}

// hideStackInitModal hides the stack init modal and pops focus
func (m *Model) hideStackInitModal() {
	m.ui.StackInitModal.Hide()
	m.ui.Focus.Remove(ui.FocusStackInitModal)
}

// showStackSelector shows the stack selector and pushes focus to it
func (m *Model) showStackSelector() {
	m.ui.StackSelector.SetLoading(true)
	m.ui.StackSelector.Show()
	m.ui.Focus.Push(ui.FocusStackSelector)
}

// hideStackSelector hides the stack selector and pops focus
func (m *Model) hideStackSelector() {
	m.ui.StackSelector.Hide()
	m.ui.Focus.Remove(ui.FocusStackSelector)
}

// showWorkspaceSelector shows the workspace selector and pushes focus to it
func (m *Model) showWorkspaceSelector() {
	m.ui.WorkspaceSelector.SetLoading(true)
	m.ui.WorkspaceSelector.Show()
	m.ui.Focus.Push(ui.FocusWorkspaceSelector)
}

// hideWorkspaceSelector hides the workspace selector and pops focus
func (m *Model) hideWorkspaceSelector() {
	m.ui.WorkspaceSelector.Hide()
	m.ui.Focus.Remove(ui.FocusWorkspaceSelector)
}

// showHelp shows the help dialog and pushes focus to it
func (m *Model) showHelp() {
	m.ui.Focus.Push(ui.FocusHelp)
}

// hideHelp hides the help dialog and pops focus
func (m *Model) hideHelp() {
	m.ui.Focus.Remove(ui.FocusHelp)
}

// showDetailsPanel shows the details panel and pushes focus to it
func (m *Model) showDetailsPanel() {
	if m.ui.ViewMode == ui.ViewHistory {
		m.ui.HistoryDetails.Show()
		m.ui.HistoryDetails.SetItem(m.ui.HistoryList.SelectedItem())
	} else {
		m.ui.Details.Show()
		m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
	}
	m.ui.Focus.Push(ui.FocusDetailsPanel)
}

// hideDetailsPanel hides the details panel and pops focus
func (m *Model) hideDetailsPanel() {
	m.ui.Details.Hide()
	m.ui.HistoryDetails.Hide()
	m.ui.Focus.Remove(ui.FocusDetailsPanel)
}

// toggleDetailsPanel toggles the details panel visibility
func (m *Model) toggleDetailsPanel() {
	if m.ui.Focus.Current() == ui.FocusDetailsPanel {
		m.hideDetailsPanel()
	} else {
		m.showDetailsPanel()
	}
}

// joinWithSeparator joins strings with a separator
func joinWithSeparator(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// placeOverlay places an overlay string at the specified x,y position on the background
func placeOverlay(x, y int, overlay, background string) string {
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, overlayLine := range overlayLines {
		bgIdx := y + i
		if bgIdx < 0 || bgIdx >= len(bgLines) {
			continue
		}

		bgLine := bgLines[bgIdx]

		// Truncate background line to x visual width and append overlay
		truncatedBg := truncateToWidth(bgLine, x)
		// Pad if needed
		currentWidth := lipgloss.Width(truncatedBg)
		if currentWidth < x {
			truncatedBg += strings.Repeat(" ", x-currentWidth)
		}

		bgLines[bgIdx] = truncatedBg + overlayLine
	}

	return strings.Join(bgLines, "\n")
}

// truncateToWidth truncates a string (which may contain ANSI codes) to the given visual width
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Use lipgloss to handle ANSI-aware truncation
	style := lipgloss.NewStyle().MaxWidth(width)
	return style.Render(s)
}
