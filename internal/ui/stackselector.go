package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StackItem represents a stack in the selector
type StackItem struct {
	Name    string
	Current bool
}

// StackSelector is a modal dialog for selecting a stack
type StackSelector struct {
	stacks  []StackItem
	cursor  int
	width   int
	height  int
	visible bool
	loading bool
	err     error
}

// NewStackSelector creates a new stack selector
func NewStackSelector() *StackSelector {
	return &StackSelector{}
}

// SetStacks sets the list of available stacks
func (s *StackSelector) SetStacks(stacks []StackItem) {
	s.stacks = stacks
	s.loading = false
	s.err = nil
	// Set cursor to current stack if found
	for i, stack := range stacks {
		if stack.Current {
			s.cursor = i
			break
		}
	}
}

// SetLoading sets the loading state
func (s *StackSelector) SetLoading(loading bool) {
	s.loading = loading
}

// SetError sets an error state
func (s *StackSelector) SetError(err error) {
	s.err = err
	s.loading = false
}

// SetSize sets the dialog dimensions for centering
func (s *StackSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Show shows the stack selector
func (s *StackSelector) Show() {
	s.visible = true
}

// Hide hides the stack selector
func (s *StackSelector) Hide() {
	s.visible = false
}

// Visible returns whether the selector is visible
func (s *StackSelector) Visible() bool {
	return s.visible
}

// SelectedStack returns the currently selected stack name
func (s *StackSelector) SelectedStack() string {
	if len(s.stacks) == 0 || s.cursor < 0 || s.cursor >= len(s.stacks) {
		return ""
	}
	return s.stacks[s.cursor].Name
}

// HasStacks returns whether any stacks are available
func (s *StackSelector) HasStacks() bool {
	return len(s.stacks) > 0
}

// Update handles key events and returns true if a stack was selected
func (s *StackSelector) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	if !s.visible {
		return false, nil
	}

	switch {
	case key.Matches(msg, Keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, Keys.Down):
		if s.cursor < len(s.stacks)-1 {
			s.cursor++
		}
	case key.Matches(msg, Keys.Home):
		s.cursor = 0
	case key.Matches(msg, Keys.End):
		s.cursor = len(s.stacks) - 1
	case msg.String() == "enter":
		if len(s.stacks) > 0 {
			s.visible = false
			return true, nil
		}
	case key.Matches(msg, Keys.Escape):
		s.visible = false
		return false, nil
	}

	return false, nil
}

// View renders the stack selector dialog
func (s *StackSelector) View() string {
	title := DialogTitleStyle.Render("Select Stack")

	var content string
	if s.loading {
		content = DimStyle.Render("Loading stacks...")
	} else if s.err != nil {
		content = ErrorStyle.Render(s.err.Error())
	} else if len(s.stacks) == 0 {
		content = DimStyle.Render("No stacks found")
	} else {
		var lines []string
		// Calculate visible range for scrolling
		maxVisible := 10
		start := 0
		end := len(s.stacks)

		if len(s.stacks) > maxVisible {
			// Center cursor in visible range
			start = s.cursor - maxVisible/2
			if start < 0 {
				start = 0
			}
			end = start + maxVisible
			if end > len(s.stacks) {
				end = len(s.stacks)
				start = end - maxVisible
			}
		}

		// Show scroll indicator at top if needed
		if start > 0 {
			lines = append(lines, ScrollIndicatorStyle.Render("  ▲ more"))
		}

		for i := start; i < end; i++ {
			stack := s.stacks[i]
			cursor := "  "
			if i == s.cursor {
				cursor = CursorStyle.Render("> ")
			}

			name := stack.Name
			if stack.Current {
				name = ValueStyle.Render(stack.Name) + DimStyle.Render(" (current)")
			} else if i == s.cursor {
				name = ValueStyle.Render(stack.Name)
			} else {
				name = DimStyle.Render(stack.Name)
			}

			lines = append(lines, cursor+name)
		}

		// Show scroll indicator at bottom if needed
		if end < len(s.stacks) {
			lines = append(lines, ScrollIndicatorStyle.Render("  ▼ more"))
		}

		content = strings.Join(lines, "\n")
	}

	// Footer hints
	footer := DimStyle.Render("\n↑/↓ navigate  enter select  esc cancel")

	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content, footer))

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorBg),
	)
}
