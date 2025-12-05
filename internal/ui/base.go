package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
)

// PanelBase provides common panel functionality for detail panels.
// Embed this in panel types to get standard visibility, scroll, and size management.
type PanelBase struct {
	visible      bool
	scrollOffset int
	width        int
	height       int
}

// Visible returns whether the panel is visible
func (p *PanelBase) Visible() bool {
	return p.visible
}

// Toggle toggles the panel visibility and resets scroll offset
func (p *PanelBase) Toggle() {
	p.visible = !p.visible
	p.scrollOffset = 0
}

// Show shows the panel and resets scroll offset
func (p *PanelBase) Show() {
	p.visible = true
	p.scrollOffset = 0
}

// Hide hides the panel
func (p *PanelBase) Hide() {
	p.visible = false
}

// SetSize sets the dimensions for the panel
func (p *PanelBase) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Width returns the panel width
func (p *PanelBase) Width() int {
	return p.width
}

// Height returns the panel height
func (p *PanelBase) Height() int {
	return p.height
}

// ScrollOffset returns the current scroll offset
func (p *PanelBase) ScrollOffset() int {
	return p.scrollOffset
}

// SetScrollOffset sets the scroll offset directly
func (p *PanelBase) SetScrollOffset(offset int) {
	p.scrollOffset = offset
}

// ScrollUp scrolls the content up by the given number of lines
func (p *PanelBase) ScrollUp(lines int) {
	p.scrollOffset -= lines
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

// ScrollDown scrolls the content down by the given number of lines
func (p *PanelBase) ScrollDown(lines int) {
	p.scrollOffset += lines
}

// ResetScroll resets the scroll offset to 0
func (p *PanelBase) ResetScroll() {
	p.scrollOffset = 0
}

// ListBase provides common list functionality for list components.
// Note: This is a partial base - lists have complex state that varies significantly.
// Use this for common loading/error state management.
type ListBase struct {
	loading    bool
	loadingMsg string
	spinner    spinner.Model
	err        error
	width      int
	height     int
	ready      bool
}

// InitSpinner initializes the spinner with default settings
func (l *ListBase) InitSpinner() {
	l.spinner = spinner.New()
	l.spinner.Spinner = spinner.Dot
}

// Spinner returns the spinner model for tick updates
func (l *ListBase) Spinner() spinner.Model {
	return l.spinner
}

// SetSpinner updates the spinner model
func (l *ListBase) SetSpinner(s spinner.Model) {
	l.spinner = s
}

// SetLoading sets the loading state with an optional message
func (l *ListBase) SetLoading(loading bool, msg string) {
	l.loading = loading
	l.loadingMsg = msg
}

// SetError sets an error state and clears loading
func (l *ListBase) SetError(err error) {
	l.err = err
	l.loading = false
}

// ClearError clears the error state
func (l *ListBase) ClearError() {
	l.err = nil
}

// IsLoading returns true if in loading state
func (l *ListBase) IsLoading() bool {
	return l.loading
}

// Error returns the current error, or nil if none
func (l *ListBase) Error() error {
	return l.err
}

// SetSize sets the dimensions for the list
func (l *ListBase) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.ready = true
}

// Width returns the list width
func (l *ListBase) Width() int {
	return l.width
}

// Height returns the list height
func (l *ListBase) Height() int {
	return l.height
}

// IsReady returns true if the list has been sized
func (l *ListBase) IsReady() bool {
	return l.ready
}

// RenderLoadingState renders the loading spinner if loading, error if error, or empty string.
// Returns (rendered string, handled bool). If handled is true, use the rendered string.
// If handled is false, render items normally.
func (l *ListBase) RenderLoadingState() (string, bool) {
	if l.loading {
		return RenderCenteredLoading(l.spinner, l.loadingMsg, l.width, l.height), true
	}
	if l.err != nil {
		return RenderPaddedError(l.err), true
	}
	return "", false
}
