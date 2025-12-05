package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// StackItem represents a stack in the selector
type StackItem struct {
	Name    string
	Current bool
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
}

// NewStackSelector creates a new stack selector
func NewStackSelector() *StackSelector {
	dialog := NewSelectorDialog[StackItem]("Select Stack")
	dialog.SetLoadingText("Loading stacks...")
	dialog.SetEmptyText("No stacks found")
	return &StackSelector{
		SelectorDialog: dialog,
	}
}

// SetStacks sets the list of available stacks
func (s *StackSelector) SetStacks(stacks []StackItem) {
	s.SetItems(stacks)
}

// SelectedStack returns the currently selected stack name
func (s *StackSelector) SelectedStack() string {
	item := s.SelectedItem()
	if item == nil {
		return ""
	}
	return item.Name
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
