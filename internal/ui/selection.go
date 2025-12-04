package ui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextSelection tracks mouse drag selection state for the details panel
type TextSelection struct {
	active   bool // Currently selecting (mouse down)
	hasRange bool // Has a valid selection range

	// Start position (where mouse was pressed)
	startX, startY int

	// End position (current or final mouse position)
	endX, endY int

	// Bounds of the selectable area (relative to terminal)
	boundsX, boundsY          int
	boundsWidth, boundsHeight int
}

// NewTextSelection creates a new text selection tracker
func NewTextSelection() *TextSelection {
	return &TextSelection{}
}

// SetBounds sets the area where selection is allowed
func (s *TextSelection) SetBounds(x, y, width, height int) {
	s.boundsX = x
	s.boundsY = y
	s.boundsWidth = width
	s.boundsHeight = height
}

// ToLocal converts terminal coordinates to local coordinates within bounds
func (s *TextSelection) ToLocal(x, y int) (localX, localY int) {
	return x - s.boundsX, y - s.boundsY
}

// StartSelection begins a new selection at the given terminal coordinates
func (s *TextSelection) StartSelection(x, y int) {
	s.active = true
	s.hasRange = false
	s.startX, s.startY = x, y
	s.endX, s.endY = x, y
}

// UpdateSelection updates the selection end point during drag
func (s *TextSelection) UpdateSelection(x, y int) {
	if !s.active {
		return
	}
	s.endX, s.endY = x, y
	s.hasRange = s.startX != s.endX || s.startY != s.endY
}

// EndSelection finalizes the selection
func (s *TextSelection) EndSelection(x, y int) {
	if !s.active {
		return
	}
	s.active = false
	s.endX, s.endY = x, y
	s.hasRange = s.startX != s.endX || s.startY != s.endY
}

// Clear removes the current selection
func (s *TextSelection) Clear() {
	s.active = false
	s.hasRange = false
}

// Active returns true if currently selecting (mouse down)
func (s *TextSelection) Active() bool {
	return s.active
}

// HasSelection returns true if there is a valid selection range
func (s *TextSelection) HasSelection() bool {
	return s.hasRange
}

// GetNormalizedRange returns the selection range normalized so start <= end
// Returns local coordinates (relative to bounds)
func (s *TextSelection) GetNormalizedRange() (startY, startX, endY, endX int) {
	// Convert to local coordinates
	startX, startY = s.ToLocal(s.startX, s.startY)
	endX, endY = s.ToLocal(s.endX, s.endY)

	// Normalize so start comes before end
	if startY > endY || (startY == endY && startX > endX) {
		startY, endY = endY, startY
		startX, endX = endX, startX
	}
	return startY, startX, endY, endX
}

// IsPositionSelected checks if a position (in local coordinates) is within the selection
func (s *TextSelection) IsPositionSelected(x, y int) bool {
	if !s.hasRange && !s.active {
		return false
	}

	startY, startX, endY, endX := s.GetNormalizedRange()

	// Outside row range
	if y < startY || y > endY {
		return false
	}

	// Single row selection
	if startY == endY {
		return x >= startX && x <= endX
	}

	// First row of multi-row selection
	if y == startY {
		return x >= startX
	}

	// Last row of multi-row selection
	if y == endY {
		return x <= endX
	}

	// Middle rows are fully selected
	return true
}

// ExtractSelectedText extracts the text within the selection from rendered lines
// lines should be the plain text content (without ANSI codes)
// Uses local coordinates (relative to bounds)
func (s *TextSelection) ExtractSelectedText(lines []string, startRow int) string {
	if !s.hasRange && !s.active {
		return ""
	}

	startY, _, endY, _ := s.GetNormalizedRange()
	var result strings.Builder

	for lineIdx, line := range lines {
		row := startRow + lineIdx
		if row < startY || row > endY {
			continue
		}

		runes := []rune(line)
		var selectedRunes []rune
		col := 0

		for _, r := range runes {
			if s.IsPositionSelected(col, row) {
				selectedRunes = append(selectedRunes, r)
			}
			col++
		}

		if len(selectedRunes) > 0 {
			if result.Len() > 0 {
				result.WriteRune('\n')
			}
			result.WriteString(string(selectedRunes))
		}
	}

	return result.String()
}

// TextSelectionStyle returns a style for mouse-selected text
var TextSelectionStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("39")). // Blue
	Foreground(lipgloss.Color("15"))  // White

// CopyToClipboardCmd returns a Bubble Tea command that copies text to clipboard
// Uses pbcopy on macOS for reliability
func CopyToClipboardCmd(text string) tea.Cmd {
	return CopyToClipboardWithCountCmd(text, 0)
}

// CopyToClipboardWithCountCmd copies text and includes resource count in the message
func CopyToClipboardWithCountCmd(text string, count int) tea.Cmd {
	if text == "" {
		return nil
	}

	return func() tea.Msg {
		// Use pbcopy on macOS (most reliable)
		if err := copyWithPbcopy(text); err == nil {
			return CopiedToClipboardMsg{Text: text, Count: count}
		}

		// Clipboard copy failed
		return CopyFailedMsg{}
	}
}

// copyWithPbcopy uses macOS pbcopy command
func copyWithPbcopy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// CopiedToClipboardMsg is sent after text has been copied
type CopiedToClipboardMsg struct {
	Text  string
	Count int // Number of resources copied (0 = single/text, >0 = multiple resources)
}

// CopyFailedMsg is sent when clipboard copy fails
type CopyFailedMsg struct{}
