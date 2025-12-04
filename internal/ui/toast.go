package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToastDuration is how long the toast is visible
const ToastDuration = 4 * time.Second

// FlashDuration is how long the highlight flash lasts
const FlashDuration = 100 * time.Millisecond

// Toast is a temporary notification message
type Toast struct {
	message   string
	visible   bool
	startTime time.Time
}

// ToastMsg triggers showing a toast
type ToastMsg struct {
	Message string
}

// ToastHideMsg hides the toast after timeout
type ToastHideMsg struct{}

// NewToast creates a new toast component
func NewToast() *Toast {
	return &Toast{}
}

// Show displays a toast message
func (t *Toast) Show(message string) tea.Cmd {
	t.message = message
	t.visible = true
	t.startTime = time.Now()

	// Return a command to hide the toast after duration
	return tea.Tick(ToastDuration, func(time.Time) tea.Msg {
		return ToastHideMsg{}
	})
}

// Hide hides the toast
func (t *Toast) Hide() {
	t.visible = false
	t.message = ""
}

// Visible returns whether the toast is visible
func (t *Toast) Visible() bool {
	return t.visible
}

// View renders the toast
func (t *Toast) View(width int) string {
	if !t.visible || t.message == "" {
		return ""
	}

	style := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 2).
		Bold(true)

	toast := style.Render(t.message)

	// Center the toast
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, toast)
}
