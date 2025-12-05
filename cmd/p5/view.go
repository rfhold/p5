package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/rfhold/p5/internal/ui"
)

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Build header
	header := m.header.View()

	// Build footer with keybind hints
	footer := m.renderFooter()

	// Calculate available height for main content
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	mainHeight := m.height - headerHeight - footerHeight - 1

	if mainHeight < 1 {
		mainHeight = 1
	}

	// Build main content area
	var mainContent string
	if m.viewMode == ui.ViewHistory {
		m.historyList.SetSize(m.width, mainHeight)
		mainContent = m.historyList.View()
	} else {
		mainContent = m.resourceList.View()
	}
	mainArea := lipgloss.NewStyle().
		Height(mainHeight).
		Width(m.width).
		Render(mainContent)

	fullView := lipgloss.JoinVertical(lipgloss.Left, header, mainArea, footer)

	// Overlay details panel on right half if visible (resource or history)
	if m.viewMode == ui.ViewHistory && m.historyDetails.Visible() {
		detailsWidth := m.width / 2
		m.historyDetails.SetSize(detailsWidth, mainHeight)
		detailsView := m.historyDetails.View()

		// Place the details panel on the right side
		fullView = placeOverlay(m.width/2, headerHeight, detailsView, fullView)
	} else if m.details.Visible() {
		detailsWidth := m.width / 2
		m.details.SetSize(detailsWidth, mainHeight)
		detailsView := m.details.View()

		// Place the details panel on the right side
		fullView = placeOverlay(m.width/2, headerHeight, detailsView, fullView)
	}

	// Overlay help dialog if showing
	if m.showHelp {
		fullView = m.help.View()
	}

	// Overlay stack selector if showing
	if m.stackSelector.Visible() {
		fullView = m.stackSelector.View()
	}

	// Overlay workspace selector if showing
	if m.workspaceSelector.Visible() {
		fullView = m.workspaceSelector.View()
	}

	// Overlay import modal if showing
	if m.importModal.Visible() {
		fullView = m.importModal.View()
	}

	// Overlay confirm modal if showing
	if m.confirmModal.Visible() {
		fullView = m.confirmModal.View()
	}

	// Overlay error modal if showing
	if m.errorModal.Visible() {
		fullView = m.errorModal.View()
	}

	// Overlay toast notification if showing
	if m.toast.Visible() {
		toastView := m.toast.View(m.width)
		// Place toast near the bottom, above the footer
		footerHeight := 1
		toastY := m.height - footerHeight - 2
		if toastY < 0 {
			toastY = 0
		}
		fullView = placeOverlay(0, toastY, toastView, fullView)
	}

	return fullView
}

// renderFooter renders the bottom footer with keybind hints
func (m Model) renderFooter() string {
	var leftParts []string
	var rightParts []string

	// Show visual mode indicator
	if m.resourceList.VisualMode() {
		leftParts = append(leftParts, ui.LabelStyle.Render("VISUAL"))
	}

	// Show flag counts if any
	if m.resourceList.HasFlags() {
		targets := len(m.resourceList.GetTargetURNs())
		replaces := len(m.resourceList.GetReplaceURNs())
		excludes := len(m.resourceList.GetExcludeURNs())

		var flagParts []string
		if targets > 0 {
			flagParts = append(flagParts, ui.FlagTargetStyle.Render(fmt.Sprintf("T:%d", targets)))
		}
		if replaces > 0 {
			flagParts = append(flagParts, ui.FlagReplaceStyle.Render(fmt.Sprintf("R:%d", replaces)))
		}
		if excludes > 0 {
			flagParts = append(flagParts, ui.FlagExcludeStyle.Render(fmt.Sprintf("E:%d", excludes)))
		}
		if len(flagParts) > 0 {
			leftParts = append(leftParts, strings.Join(flagParts, " "))
			leftParts = append(leftParts, ui.DimStyle.Render("C clear all"))
		}
	}

	// Keybind hints on the right - context sensitive
	if m.resourceList.VisualMode() {
		rightParts = append(rightParts, ui.DimStyle.Render("T target"))
		rightParts = append(rightParts, ui.DimStyle.Render("R replace"))
		rightParts = append(rightParts, ui.DimStyle.Render("E exclude"))
		rightParts = append(rightParts, ui.DimStyle.Render("esc cancel"))
	} else {
		// Show operation hints based on view
		switch m.viewMode {
		case ui.ViewStack:
			rightParts = append(rightParts, ui.DimStyle.Render("u up"))
			rightParts = append(rightParts, ui.DimStyle.Render("r refresh"))
			rightParts = append(rightParts, ui.DimStyle.Render("d destroy"))
			rightParts = append(rightParts, ui.DimStyle.Render("x delete"))
		case ui.ViewPreview:
			rightParts = append(rightParts, ui.DimStyle.Render("ctrl+u execute"))
			rightParts = append(rightParts, ui.DimStyle.Render("I import"))
			rightParts = append(rightParts, ui.DimStyle.Render("esc back"))
		case ui.ViewExecute:
			rightParts = append(rightParts, ui.DimStyle.Render("esc cancel"))
		case ui.ViewHistory:
			rightParts = append(rightParts, ui.DimStyle.Render("esc back"))
		}
		rightParts = append(rightParts, ui.DimStyle.Render("v select"))
		rightParts = append(rightParts, ui.DimStyle.Render("D details"))
		rightParts = append(rightParts, ui.DimStyle.Render("s stack"))
		rightParts = append(rightParts, ui.DimStyle.Render("w workspace"))
		rightParts = append(rightParts, ui.DimStyle.Render("h history"))
		rightParts = append(rightParts, ui.DimStyle.Render("? help"))
		rightParts = append(rightParts, ui.DimStyle.Render("q quit"))
	}

	left := joinWithSeparator(leftParts, "  ")
	right := joinWithSeparator(rightParts, "  ")

	// Calculate padding between left and right
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := m.width - leftWidth - rightWidth - 2 // -2 for margins
	if padding < 1 {
		padding = 1
	}

	return " " + left + strings.Repeat(" ", padding) + right + " "
}
