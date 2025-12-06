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

	header := m.ui.Header.View()
	footer := m.renderFooter()

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	mainHeight := m.ui.Height - headerHeight - footerHeight - 1

	if mainHeight < 1 {
		mainHeight = 1
	}

	var mainContent string
	if m.ui.ViewMode == ui.ViewHistory {
		m.ui.HistoryList.SetSize(m.ui.Width, mainHeight)
		mainContent = m.ui.HistoryList.View()
	} else {
		mainContent = m.ui.ResourceList.View()
	}
	mainArea := lipgloss.NewStyle().
		Height(mainHeight).
		Width(m.ui.Width).
		Render(mainContent)

	fullView := lipgloss.JoinVertical(lipgloss.Left, header, mainArea, footer)

	if m.ui.Focus.Has(ui.FocusDetailsPanel) {
		detailsWidth := m.ui.Width / 2
		if m.ui.ViewMode == ui.ViewHistory {
			m.ui.HistoryDetails.SetSize(detailsWidth, mainHeight)
			fullView = placeOverlay(m.ui.Width/2, headerHeight, m.ui.HistoryDetails.View(), fullView)
		} else {
			m.ui.Details.SetSize(detailsWidth, mainHeight)
			fullView = placeOverlay(m.ui.Width/2, headerHeight, m.ui.Details.View(), fullView)
		}
	}

	if m.ui.Focus.Has(ui.FocusHelp) {
		fullView = m.ui.Help.View()
	}

	if m.ui.StackSelector.Visible() {
		fullView = m.ui.StackSelector.View()
	}

	if m.ui.WorkspaceSelector.Visible() {
		fullView = m.ui.WorkspaceSelector.View()
	}

	if m.ui.ImportModal.Visible() {
		fullView = m.ui.ImportModal.View()
	}

	if m.ui.StackInitModal.Visible() {
		fullView = m.ui.StackInitModal.View()
	}

	if m.ui.ConfirmModal.Visible() {
		fullView = m.ui.ConfirmModal.View()
	}

	if m.ui.ErrorModal.Visible() {
		fullView = m.ui.ErrorModal.View()
	}

	if m.ui.Toast.Visible() {
		toastView := m.ui.Toast.View(m.ui.Width)
		footerHeight := 1
		toastY := m.ui.Height - footerHeight - 2
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

	if m.ui.ResourceList.VisualMode() {
		leftParts = append(leftParts, ui.LabelStyle.Render("VISUAL"))
	}

	if m.ui.ResourceList.HasFlags() {
		targets := len(m.ui.ResourceList.GetTargetURNs())
		replaces := len(m.ui.ResourceList.GetReplaceURNs())
		excludes := len(m.ui.ResourceList.GetExcludeURNs())

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

	if m.ui.ResourceList.VisualMode() {
		rightParts = append(rightParts, ui.DimStyle.Render("T target"))
		rightParts = append(rightParts, ui.DimStyle.Render("R replace"))
		rightParts = append(rightParts, ui.DimStyle.Render("E exclude"))
		rightParts = append(rightParts, ui.DimStyle.Render("esc cancel"))
	} else {
		switch m.ui.ViewMode {
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

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := m.ui.Width - leftWidth - rightWidth - 2 // -2 for margins
	if padding < 1 {
		padding = 1
	}

	return " " + left + strings.Repeat(" ", padding) + right + " "
}
