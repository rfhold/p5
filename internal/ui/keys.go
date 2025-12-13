package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all application keybindings
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Selection flags (uppercase)
	ToggleTarget  key.Binding
	ToggleReplace key.Binding
	ToggleExclude key.Binding
	ClearFlags    key.Binding
	ClearAllFlags key.Binding

	// Visual mode
	VisualMode key.Binding
	Escape     key.Binding

	// Operations - Preview (lowercase)
	PreviewUp      key.Binding
	PreviewRefresh key.Binding
	PreviewDestroy key.Binding

	// Operations - Execute (ctrl+key)
	ExecuteUp      key.Binding
	ExecuteRefresh key.Binding
	ExecuteDestroy key.Binding

	// Copy resource
	CopyResource     key.Binding
	CopyAllResources key.Binding

	// Details panel
	ToggleDetails key.Binding

	// Stack selector
	SelectStack key.Binding

	// Workspace selector
	SelectWorkspace key.Binding

	// History view
	ViewHistory key.Binding

	// Import
	Import key.Binding

	// Delete from state
	DeleteFromState key.Binding

	// Toggle protection
	ToggleProtect key.Binding

	// Open resource
	OpenResource key.Binding

	// Filter
	Filter key.Binding

	// General
	Help key.Binding
	Quit key.Binding
}

// Keys is the default keybinding configuration
var Keys = KeyMap{
	// Navigation
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+b"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+f"),
		key.WithHelp("pgdn", "page down"),
	),
	Home: key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("g", "top"),
	),
	End: key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("G", "bottom"),
	),

	// Selection flags (uppercase)
	ToggleTarget: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "toggle target"),
	),
	ToggleReplace: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "toggle replace"),
	),
	ToggleExclude: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "toggle exclude"),
	),
	ClearFlags: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear flags"),
	),
	ClearAllFlags: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "clear all flags"),
	),

	// Visual mode
	VisualMode: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "visual select"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),

	// Operations - Preview (lowercase)
	PreviewUp: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "preview up"),
	),
	PreviewRefresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "preview refresh"),
	),
	PreviewDestroy: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "preview destroy"),
	),

	// Operations - Execute (ctrl+key)
	ExecuteUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "execute up"),
	),
	ExecuteRefresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "execute refresh"),
	),
	ExecuteDestroy: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "execute destroy"),
	),

	// Copy resource
	CopyResource: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy resource JSON"),
	),
	CopyAllResources: key.NewBinding(
		key.WithKeys("Y"),
		key.WithHelp("Y", "copy all resources JSON"),
	),

	// Details panel
	ToggleDetails: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "toggle details"),
	),

	// Stack selector
	SelectStack: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "select stack"),
	),

	// Workspace selector
	SelectWorkspace: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "select workspace"),
	),

	// History view
	ViewHistory: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "view history"),
	),

	// Import
	Import: key.NewBinding(
		key.WithKeys("I"),
		key.WithHelp("I", "import resource"),
	),

	// Delete from state
	DeleteFromState: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete from state"),
	),

	// Toggle protection
	ToggleProtect: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "toggle protect"),
	),

	// Open resource
	OpenResource: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open resource"),
	),

	// Filter
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),

	// General
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// ShortHelp returns keybindings for the short help view
func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the full help view
func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Home, k.End},
		{k.VisualMode, k.Escape},
		{k.ToggleTarget, k.ToggleReplace, k.ToggleExclude, k.ClearFlags, k.ClearAllFlags},
		{k.PreviewUp, k.PreviewRefresh, k.PreviewDestroy},
		{k.ExecuteUp, k.ExecuteRefresh, k.ExecuteDestroy},
		{k.CopyResource, k.ToggleDetails, k.SelectStack, k.SelectWorkspace, k.ViewHistory},
		{k.Import, k.DeleteFromState, k.ToggleProtect, k.OpenResource},
		{k.Help, k.Quit},
	}
}
