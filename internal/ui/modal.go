package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ModalBase provides common functionality for modal dialogs
type ModalBase struct {
	visible      bool
	width        int
	height       int
	scrollOffset int
}

// SetSize sets the modal dimensions for centering
func (m *ModalBase) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Show shows the modal
func (m *ModalBase) Show() {
	m.visible = true
	m.scrollOffset = 0
}

// Hide hides the modal
func (m *ModalBase) Hide() {
	m.visible = false
}

// Visible returns whether the modal is visible
func (m *ModalBase) Visible() bool {
	return m.visible
}

// Toggle toggles the modal visibility
func (m *ModalBase) Toggle() {
	m.visible = !m.visible
}

// Width returns the modal width
func (m *ModalBase) Width() int {
	return m.width
}

// Height returns the modal height
func (m *ModalBase) Height() int {
	return m.height
}

// ScrollOffset returns the current scroll offset
func (m *ModalBase) ScrollOffset() int {
	return m.scrollOffset
}

// SetScrollOffset sets the scroll offset directly
func (m *ModalBase) SetScrollOffset(offset int) {
	m.scrollOffset = offset
}

// ScrollUp scrolls the content up by the given number of lines
func (m *ModalBase) ScrollUp(lines int) {
	m.scrollOffset -= lines
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// ScrollDown scrolls the content down by the given number of lines
func (m *ModalBase) ScrollDown(lines int) {
	m.scrollOffset += lines
}

// ResetScroll resets the scroll offset to 0
func (m *ModalBase) ResetScroll() {
	m.scrollOffset = 0
}

// CenterDialog centers the given dialog content on screen
func (m *ModalBase) CenterDialog(dialog string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorBg),
	)
}

// RenderDialog creates a dialog with title, content, and footer, then centers it
func (m *ModalBase) RenderDialog(title, content, footer string) string {
	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content, footer))
	return m.CenterDialog(dialog)
}

// RenderDialogWithStyle creates a dialog with custom style, then centers it
func (m *ModalBase) RenderDialogWithStyle(style lipgloss.Style, parts ...string) string {
	dialog := style.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
	return m.CenterDialog(dialog)
}

// ScrollableDialogContent holds parameters for rendering scrollable dialog content
type ScrollableDialogContent struct {
	Title        string
	Content      string
	Footer       string
	MaxHeight    int // Maximum content height before scrolling kicks in
	ScrollOffset int // Current scroll position
	DialogWidth  int // Width of the dialog
}

// ScrollableDialogResult holds the result of rendering scrollable content
type ScrollableDialogResult struct {
	Rendered        string // The fully rendered dialog
	NewScrollOffset int    // Adjusted scroll offset (clamped to valid range)
	TotalLines      int    // Total content lines
	VisibleLines    int    // Number of visible lines
	CanScroll       bool   // Whether scrolling is available
}

// RenderScrollableDialog renders a dialog with scrollable content
func (m *ModalBase) RenderScrollableDialog(params ScrollableDialogContent) ScrollableDialogResult {
	contentLines := strings.Split(params.Content, "\n")
	totalLines := len(contentLines)
	maxHeight := params.MaxHeight
	if maxHeight <= 0 {
		maxHeight = 20 // default max height
	}

	// Determine if we need scrolling
	canScroll := totalLines > maxHeight
	visibleLines := totalLines
	scrollOffset := params.ScrollOffset

	var visibleContent string
	if canScroll {
		visibleLines = maxHeight
		// Clamp scroll offset
		maxOffset := totalLines - maxHeight
		if scrollOffset > maxOffset {
			scrollOffset = maxOffset
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}

		// Get visible portion
		endIdx := scrollOffset + maxHeight
		if endIdx > totalLines {
			endIdx = totalLines
		}
		visibleContent = strings.Join(contentLines[scrollOffset:endIdx], "\n")
	} else {
		scrollOffset = 0
		visibleContent = params.Content
	}

	// Build title with scroll indicator if needed
	title := params.Title
	if canScroll {
		scrollInfo := DimStyle.Render(fmt.Sprintf(" [%d-%d/%d]", scrollOffset+1, scrollOffset+visibleLines, totalLines))
		title = title + scrollInfo
	}

	// Build footer with scroll hints if needed using unified function
	footer := params.Footer
	if canScroll {
		canScrollUp := scrollOffset > 0
		canScrollDown := scrollOffset < totalLines-maxHeight
		scrollHint := RenderScrollHint(canScrollUp, canScrollDown, "")
		if scrollHint != "" {
			footer = scrollHint + "  " + footer
		}
	}

	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, visibleContent, footer))

	return ScrollableDialogResult{
		Rendered:        m.CenterDialog(dialog),
		NewScrollOffset: scrollOffset,
		TotalLines:      totalLines,
		VisibleLines:    visibleLines,
		CanScroll:       canScroll,
	}
}
