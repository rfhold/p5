package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmModal is a reusable confirmation dialog with keybind actions
type ConfirmModal struct {
	ModalBase // Embedded modal base for common functionality

	// Dialog content
	title   string
	message string
	warning string // Optional warning text shown in red

	// Keybind labels
	confirmLabel string
	cancelLabel  string
	confirmKey   string // Key to press for confirm (default "y")
	cancelKey    string // Key to press for cancel (default "n")

	// Context data (passed through on confirm)
	contextURN  string
	contextName string
	contextType string

	// Bulk context data (for multi-resource operations)
	bulkResources []SelectedResource
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal() *ConfirmModal {
	return &ConfirmModal{
		cancelLabel:  "Cancel",
		confirmLabel: "Confirm",
		confirmKey:   "y",
		cancelKey:    "n",
	}
}

// SetSize sets the dialog dimensions for centering (delegates to ModalBase)

// Show shows the confirmation modal with the given content
func (m *ConfirmModal) Show(title, message, warning string) {
	m.title = title
	m.message = message
	m.warning = warning
	m.ModalBase.Show()
}

// ShowWithContext shows the modal and stores context data
func (m *ConfirmModal) ShowWithContext(title, message, warning, contextURN, contextName, contextType string) {
	m.Show(title, message, warning)
	m.contextURN = contextURN
	m.contextName = contextName
	m.contextType = contextType
}

// SetLabels customizes the action labels
func (m *ConfirmModal) SetLabels(cancel, confirm string) {
	m.cancelLabel = cancel
	m.confirmLabel = confirm
}

// SetKeys customizes the keybinds (default: y to confirm, n to cancel)
func (m *ConfirmModal) SetKeys(cancel, confirm string) {
	m.cancelKey = cancel
	m.confirmKey = confirm
}

// Hide hides the confirmation modal and clears context
func (m *ConfirmModal) Hide() {
	m.ModalBase.Hide()
	m.contextURN = ""
	m.contextName = ""
	m.contextType = ""
	m.bulkResources = nil
}

// ShowBulkWithContext shows the modal for bulk operations with multiple resources
func (m *ConfirmModal) ShowBulkWithContext(title, message, warning string, resources []SelectedResource) {
	m.title = title
	m.message = message
	m.warning = warning
	m.bulkResources = resources
	// Clear single-resource context
	m.contextURN = ""
	m.contextName = ""
	m.contextType = ""
	m.ModalBase.Show()
}

// GetBulkResources returns the stored bulk resources (nil for single-resource operations)
func (m *ConfirmModal) GetBulkResources() []SelectedResource {
	return m.bulkResources
}

// IsBulkOperation returns true if this is a bulk operation with multiple resources
func (m *ConfirmModal) IsBulkOperation() bool {
	return len(m.bulkResources) > 0
}

// Visible is inherited from ModalBase

// GetContextURN returns the stored context URN
func (m *ConfirmModal) GetContextURN() string {
	return m.contextURN
}

// GetContextName returns the stored context name
func (m *ConfirmModal) GetContextName() string {
	return m.contextName
}

// GetContextType returns the stored context type
func (m *ConfirmModal) GetContextType() string {
	return m.contextType
}

// Update handles key events and returns confirmation status and any tea command.
func (m *ConfirmModal) Update(msg tea.KeyMsg) (confirmed, cancelled bool, cmd tea.Cmd) {
	if !m.Visible() {
		return false, false, nil
	}

	switch {
	case msg.String() == m.confirmKey:
		m.ModalBase.Hide()
		return true, false, nil // Confirmed

	case msg.String() == m.cancelKey, key.Matches(msg, Keys.Escape):
		m.ModalBase.Hide()
		return false, true, nil // Cancelled
	}

	return false, false, nil
}

// View renders the confirmation modal
func (m *ConfirmModal) View() string {
	title := DialogTitleStyle.Render(m.title)

	// Build content
	content := ValueStyle.Render(m.message)

	// Add warning if present
	if m.warning != "" {
		content += "\n\n" + ErrorStyle.Render(m.warning)
	}

	// Footer hints showing keybinds
	footer := DimStyle.Render("\n" + m.confirmKey + " " + m.confirmLabel + "  " + m.cancelKey + "/" + "esc " + m.cancelLabel)

	return m.RenderDialog(title, content, footer)
}
