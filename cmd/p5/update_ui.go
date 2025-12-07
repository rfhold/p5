package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rfhold/p5/internal/ui"
)

// UI handlers - handles window size, spinner, toast, and clipboard

// handleWindowSize handles terminal resize events
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.ui.Width = msg.Width
	m.ui.Height = msg.Height
	m.ui.Header.SetWidth(msg.Width)
	m.ui.Help.SetSize(msg.Width, msg.Height)
	m.ui.StackSelector.SetSize(msg.Width, msg.Height)
	m.ui.WorkspaceSelector.SetSize(msg.Width, msg.Height)
	m.ui.ImportModal.SetSize(msg.Width, msg.Height)
	m.ui.ConfirmModal.SetSize(msg.Width, msg.Height)
	m.ui.ErrorModal.SetSize(msg.Width, msg.Height)
	m.ui.StackInitModal.SetSize(msg.Width, msg.Height)
	// Calculate resource list area height
	headerHeight := lipgloss.Height(m.ui.Header.View())
	footerHeight := 1 // single line footer
	listHeight := msg.Height - headerHeight - footerHeight - 1
	listHeight = max(listHeight, 1)
	m.ui.ResourceList.SetSize(msg.Width, listHeight)
	// Details panel will be sized when rendered as overlay
	detailsWidth := msg.Width / 2
	m.ui.Details.SetSize(detailsWidth, listHeight)
	return m, nil
}

// handleMouseEvent handles mouse events
func (m Model) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

// handleSpinnerTick handles spinner animation ticks
func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if m.ui.Header.IsLoading() {
		s, cmd := m.ui.Header.Spinner().Update(msg)
		m.ui.Header.SetSpinner(s)
		cmds = append(cmds, cmd)
	}
	if m.ui.ResourceList.IsLoading() {
		s, cmd := m.ui.ResourceList.Spinner().Update(msg)
		m.ui.ResourceList.SetSpinner(s)
		cmds = append(cmds, cmd)
	}
	if m.ui.HistoryList.IsLoading() {
		s, cmd := m.ui.HistoryList.Spinner().Update(msg)
		m.ui.HistoryList.SetSpinner(s)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// handleCopiedToClipboard handles clipboard copy confirmation
func (m Model) handleCopiedToClipboard(msg ui.CopiedToClipboardMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Get selected item name for single resource copy
	var selectedItemName string
	if msg.Count == 1 {
		if item := m.ui.ResourceList.SelectedItem(); item != nil {
			selectedItemName = item.Name
		}
	}

	toastMsg := FormatClipboardMessage(msg.Count, selectedItemName)

	// Flash clear after short duration (for both single and all)
	if msg.Count >= 1 {
		cmds = append(cmds, tea.Tick(ui.FlashDuration, func(time.Time) tea.Msg {
			return ui.FlashClearMsg{}
		}))
	}

	cmds = append(cmds, m.ui.Toast.Show(toastMsg))
	return m, tea.Batch(cmds...)
}

// handleToastHide handles toast hide event
func (m Model) handleToastHide() (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	m.ui.Toast.Hide()
	return m, nil
}

// handleFlashClear handles clearing the flash highlight
func (m Model) handleFlashClear() (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	m.ui.ResourceList.ClearFlash()
	return m, nil
}
