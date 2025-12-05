package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StackItem represents a stack in the selector
type StackItem struct {
	Name      string
	Current   bool
	IsNewItem bool // Special flag for "create new stack" option
}

// Label implements SelectorItem
func (s StackItem) Label() string {
	return s.Name
}

// IsCurrent implements SelectorItem
func (s StackItem) IsCurrent() bool {
	return s.Current
}

// StackSelector is a modal dialog for selecting a stack
type StackSelector struct {
	*SelectorDialog[StackItem]
	showNewOption bool
}

// NewStackSelector creates a new stack selector
func NewStackSelector() *StackSelector {
	dialog := NewSelectorDialog[StackItem]("Select Stack")
	dialog.SetLoadingText("Loading stacks...")
	dialog.SetEmptyText("No stacks found")

	// Custom renderer for stack items
	dialog.SetItemRenderer(func(item StackItem, isCursor bool) string {
		cursor := "  "
		if isCursor {
			cursor = CursorStyle.Render("> ")
		}

		if item.IsNewItem {
			// Style the "new stack" option distinctly (green for creation)
			newStyle := lipgloss.NewStyle().Foreground(ColorCreate)
			if isCursor {
				return cursor + newStyle.Render(item.Name)
			}
			return cursor + DimStyle.Render(item.Name)
		}

		// Regular stack items
		var name string
		if item.Current {
			name = ValueStyle.Render(item.Name) + DimStyle.Render(" (current)")
		} else if isCursor {
			name = ValueStyle.Render(item.Name)
		} else {
			name = DimStyle.Render(item.Name)
		}
		return cursor + name
	})

	return &StackSelector{
		SelectorDialog: dialog,
		showNewOption:  true, // Show "new stack" option by default
	}
}

// SetShowNewOption controls whether the "new stack" option is shown
func (s *StackSelector) SetShowNewOption(show bool) {
	s.showNewOption = show
}

// SetStacks sets the list of available stacks
func (s *StackSelector) SetStacks(stacks []StackItem) {
	// Prepend "new stack" option if enabled
	if s.showNewOption {
		items := make([]StackItem, 0, len(stacks)+1)
		items = append(items, StackItem{
			Name:      "+ New Stack",
			IsNewItem: true,
		})
		items = append(items, stacks...)
		s.SetItems(items)
	} else {
		s.SetItems(stacks)
	}
}

// SelectedStack returns the currently selected stack name
// Returns empty string if "new stack" option is selected
func (s *StackSelector) SelectedStack() string {
	item := s.SelectedItem()
	if item == nil || item.IsNewItem {
		return ""
	}
	return item.Name
}

// IsNewStackSelected returns true if the "new stack" option is selected
func (s *StackSelector) IsNewStackSelected() bool {
	item := s.SelectedItem()
	return item != nil && item.IsNewItem
}

// HasStacks returns whether any stacks are available
func (s *StackSelector) HasStacks() bool {
	return s.HasItems()
}

// Update handles key events and returns true if a stack was selected
func (s *StackSelector) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	return s.SelectorDialog.Update(msg)
}

// View renders the stack selector dialog
func (s *StackSelector) View() string {
	return s.SelectorDialog.View()
}
