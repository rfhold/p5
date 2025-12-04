package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorkspaceItem represents a workspace in the selector
type WorkspaceItem struct {
	Path         string
	RelativePath string // Path relative to current working directory
	Name         string
	Current      bool
}

// WorkspaceSelector is a modal dialog for selecting a workspace
type WorkspaceSelector struct {
	workspaces []WorkspaceItem
	cursor     int
	width      int
	height     int
	visible    bool
	loading    bool
	err        error
}

// NewWorkspaceSelector creates a new workspace selector
func NewWorkspaceSelector() *WorkspaceSelector {
	return &WorkspaceSelector{}
}

// SetWorkspaces sets the list of available workspaces
func (s *WorkspaceSelector) SetWorkspaces(workspaces []WorkspaceItem) {
	s.workspaces = workspaces
	s.loading = false
	s.err = nil
	// Set cursor to current workspace if found
	for i, ws := range workspaces {
		if ws.Current {
			s.cursor = i
			break
		}
	}
}

// SetLoading sets the loading state
func (s *WorkspaceSelector) SetLoading(loading bool) {
	s.loading = loading
}

// SetError sets an error state
func (s *WorkspaceSelector) SetError(err error) {
	s.err = err
	s.loading = false
}

// SetSize sets the dialog dimensions for centering
func (s *WorkspaceSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Show shows the workspace selector
func (s *WorkspaceSelector) Show() {
	s.visible = true
}

// Hide hides the workspace selector
func (s *WorkspaceSelector) Hide() {
	s.visible = false
}

// Visible returns whether the selector is visible
func (s *WorkspaceSelector) Visible() bool {
	return s.visible
}

// SelectedWorkspace returns the currently selected workspace
func (s *WorkspaceSelector) SelectedWorkspace() *WorkspaceItem {
	if len(s.workspaces) == 0 || s.cursor < 0 || s.cursor >= len(s.workspaces) {
		return nil
	}
	return &s.workspaces[s.cursor]
}

// HasWorkspaces returns whether any workspaces are available
func (s *WorkspaceSelector) HasWorkspaces() bool {
	return len(s.workspaces) > 0
}

// Update handles key events and returns true if a workspace was selected
func (s *WorkspaceSelector) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	if !s.visible {
		return false, nil
	}

	switch {
	case key.Matches(msg, Keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, Keys.Down):
		if s.cursor < len(s.workspaces)-1 {
			s.cursor++
		}
	case key.Matches(msg, Keys.Home):
		s.cursor = 0
	case key.Matches(msg, Keys.End):
		s.cursor = len(s.workspaces) - 1
	case msg.String() == "enter":
		if len(s.workspaces) > 0 {
			s.visible = false
			return true, nil
		}
	case key.Matches(msg, Keys.Escape):
		s.visible = false
		return false, nil
	}

	return false, nil
}

// View renders the workspace selector dialog
func (s *WorkspaceSelector) View() string {
	title := DialogTitleStyle.Render("Select Workspace")

	var content string
	if s.loading {
		content = DimStyle.Render("Searching for Pulumi projects...")
	} else if s.err != nil {
		content = ErrorStyle.Render(s.err.Error())
	} else if len(s.workspaces) == 0 {
		content = DimStyle.Render("No Pulumi projects found")
	} else {
		var lines []string
		// Calculate visible range for scrolling
		maxVisible := 10
		start := 0
		end := len(s.workspaces)

		if len(s.workspaces) > maxVisible {
			// Center cursor in visible range
			start = s.cursor - maxVisible/2
			if start < 0 {
				start = 0
			}
			end = start + maxVisible
			if end > len(s.workspaces) {
				end = len(s.workspaces)
				start = end - maxVisible
			}
		}

		// Show scroll indicator at top if needed
		if start > 0 {
			lines = append(lines, ScrollIndicatorStyle.Render("  ▲ more"))
		}

		for i := start; i < end; i++ {
			ws := s.workspaces[i]
			cursor := "  "
			if i == s.cursor {
				cursor = CursorStyle.Render("> ")
			}

			// Format: name (path) - use relative path if available
			name := ws.Name
			displayPath := ws.RelativePath
			if displayPath == "" {
				displayPath = ws.Path
			}
			path := DimStyle.Render(" " + displayPath)

			if ws.Current {
				name = ValueStyle.Render(ws.Name) + DimStyle.Render(" (current)")
				path = ""
			} else if i == s.cursor {
				name = ValueStyle.Render(ws.Name)
			} else {
				name = DimStyle.Render(ws.Name)
			}

			lines = append(lines, cursor+name+path)
		}

		// Show scroll indicator at bottom if needed
		if end < len(s.workspaces) {
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
