# Bubbletea

p5 uses Bubbletea for the terminal UI framework.

## Package

```go
import tea "github.com/charmbracelet/bubbletea"
```

## Architecture

### Model-Update-View (MVU)

```
cmd/p5/
  model.go       - Main Model struct with AppState and UIState
  view.go        - Rendering logic
  update_*.go    - Message handlers by domain
  messages.go    - Custom message types
  commands.go    - Async tea.Cmd functions
```

### State Separation

- `AppState`: Pure application state (init state, operation state, flags)
- `UIState`: UI component state (layout, focus, lists, modals)

## Update Routing

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg: return m.handleWindowSize(msg)
    case tea.MouseMsg:      return m.handleMouseEvent(msg)
    case tea.KeyMsg:        return m.handleKeyPress(msg)
    default:                return m.handleMessage(msg)
    }
}
```

Update handlers are split by concern:
- `update_keys.go` - Keyboard input with focus-aware routing
- `update_messages.go` - Async message dispatch
- `update_init.go` - Initialization state machine
- `update_operations.go` - Preview/execute events
- `update_selection.go` - Stack/workspace selection
- `update_ui.go` - Window size, spinner, toast, clipboard

## Components

Located in `internal/ui/`:

### Base Types
- `PanelBase` - Visibility, scroll, size for panels
- `ListBase` - Loading state, spinner, error for lists
- `ModalBase` - Visibility, centering for dialogs
- `SelectorDialog[T]` - Generic selection with filtering

### Specialized Components
- `ResourceList` - Tree-organized resource list with visual selection
- `ResourceTree` - Tree rendering logic
- `Header` - Program/stack info with operation summary
- `HistoryList` / `HistoryDetails` - Update history display
- Modals: `ImportModal`, `ConfirmModal`, `ErrorModal`, `StackInitModal`
- Selectors: `StackSelector`, `WorkspaceSelector`

## Focus Management

Stack-based focus system in `internal/ui/focus.go`:

```go
const (
    FocusMain              // Normal interaction
    FocusDetailsPanel      // Details panel scroll
    FocusHelp              // Help dialog
    FocusStackSelector     // Stack modal
    FocusWorkspaceSelector // Workspace modal
    FocusImportModal       // Import modal
    FocusStackInitModal    // Stack creation
    FocusConfirmModal      // Confirmation
    FocusErrorModal        // Error (highest)
)
```

## Message Patterns

### Async Commands
```go
func (m Model) fetchStacksList() tea.Cmd {
    return func() tea.Msg {
        stacks, files, err := m.deps.Workspace.ListStacks(...)
        if err != nil { return errMsg(err) }
        return stacksListMsg{Stacks: stacks, Files: files}
    }
}
```

### Event Streaming
```go
m.previewCh = msg.ch
return m, waitForPreviewEvent(m.previewCh)
```

### Batching
```go
return m, tea.Batch(m.fetchProjectInfo(), m.authenticatePlugins())
```

## Key Bindings

Defined in `internal/ui/keys.go` using `bubbles/key` package. Keybindings are context-aware based on current focus layer and view mode.
