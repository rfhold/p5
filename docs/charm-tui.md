# Charm TUI Libraries Reference

Reference documentation for building TUI applications with Bubble Tea, Bubbles, and Lip Gloss.

## Overview

- **Bubble Tea** - Core framework using Elm architecture (Model-Update-View)
- **Bubbles** - Pre-built UI components (inputs, lists, spinners, etc.)
- **Lip Gloss** - Styling and layout library

## Bubble Tea Architecture

### The Elm Architecture

```go
// Model - stores application state
type model struct {
    items    []string
    cursor   int
    selected map[int]struct{}
}

// Init - returns initial command (or nil)
func (m model) Init() tea.Cmd {
    return nil
}

// Update - handles messages, returns updated model and optional command
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case "up":
            m.cursor--
        case "down":
            m.cursor++
        }
    }
    return m, nil
}

// View - renders UI as string
func (m model) View() string {
    var s strings.Builder
    for i, item := range m.items {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        s.WriteString(fmt.Sprintf("%s %s\n", cursor, item))
    }
    return s.String()
}
```

### Program Lifecycle

```go
func main() {
    p := tea.NewProgram(
        initialModel(),
        tea.WithAltScreen(),       // Full screen mode
        tea.WithMouseCellMotion(), // Mouse support
    )
    
    finalModel, err := p.Run()
    if err != nil {
        log.Fatal(err)
    }
}
```

### Built-in Message Types

| Type | Description |
|------|-------------|
| `tea.KeyMsg` | Keyboard input |
| `tea.MouseMsg` | Mouse events |
| `tea.WindowSizeMsg` | Terminal resize (has `.Width`, `.Height`) |
| `tea.FocusMsg` | Terminal gained focus |
| `tea.BlurMsg` | Terminal lost focus |

## Commands (Async Operations)

Commands are the ONLY way to do I/O. Never use goroutines directly.

```go
// A Cmd is a function that returns a Msg
type Cmd func() Msg

// Simple command
func fetchData() tea.Msg {
    data, err := api.Fetch()
    if err != nil {
        return errMsg{err}
    }
    return dataMsg{data}
}

// Command with arguments - return a function that returns Cmd
func fetchUser(id int) tea.Cmd {
    return func() tea.Msg {
        user, err := api.GetUser(id)
        if err != nil {
            return errMsg{err}
        }
        return userMsg{user}
    }
}

// In Init or Update, return commands:
func (m model) Init() tea.Cmd {
    return tea.Batch(
        m.spinner.Tick,  // Start spinner
        fetchData,       // Fetch data
    )
}
```

### Command Utilities

```go
// Run commands concurrently
tea.Batch(cmd1, cmd2, cmd3)

// Run commands sequentially
tea.Sequence(cmd1, cmd2, cmd3)

// Quit the program
tea.Quit
```

## State Management Patterns

### Multi-Screen Navigation (Root Model Pattern)

```go
type Screen int

const (
    ScreenHome Screen = iota
    ScreenList
    ScreenDetail
)

type App struct {
    currentScreen Screen
    screenStack   []Screen  // For back navigation
    
    // Screen models
    home   *HomeModel
    list   *ListModel
    detail *DetailModel
    
    // Shared state
    width  int
    height int
}

// Navigation messages
type NavigateMsg struct{ Screen Screen }
type BackMsg struct{}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        a.width = msg.Width
        a.height = msg.Height
        
    case NavigateMsg:
        a.screenStack = append(a.screenStack, a.currentScreen)
        a.currentScreen = msg.Screen
        return a, a.currentScreenModel().Init()
        
    case BackMsg:
        if len(a.screenStack) > 0 {
            a.currentScreen = a.screenStack[len(a.screenStack)-1]
            a.screenStack = a.screenStack[:len(a.screenStack)-1]
        }
        return a, nil
    }
    
    // Delegate to current screen
    return a.updateCurrentScreen(msg)
}

func (a *App) currentScreenModel() tea.Model {
    switch a.currentScreen {
    case ScreenList:
        return a.list
    case ScreenDetail:
        return a.detail
    default:
        return a.home
    }
}
```

### Parent-Child Communication

```go
// Child sends message to parent
func (m *ChildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
        return m, func() tea.Msg {
            return ItemSelectedMsg{ID: m.selected}
        }
    }
    return m, nil
}

// Parent handles child message
func (m *ParentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ItemSelectedMsg:
        m.handleSelection(msg.ID)
        return m, nil
    }
    
    // Delegate other messages to child
    var cmd tea.Cmd
    m.child, cmd = m.child.Update(msg)
    return m, cmd
}
```

## Keybindings

### Using bubbles/key

```go
import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
    Up    key.Binding
    Down  key.Binding
    Enter key.Binding
    Back  key.Binding
    Help  key.Binding
    Quit  key.Binding
}

var DefaultKeys = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("k", "up"),
        key.WithHelp("↑/k", "up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("j", "down"),
        key.WithHelp("↓/j", "down"),
    ),
    Enter: key.NewBinding(
        key.WithKeys("enter"),
        key.WithHelp("enter", "select"),
    ),
    Back: key.NewBinding(
        key.WithKeys("esc", "backspace"),
        key.WithHelp("esc", "back"),
    ),
    Help: key.NewBinding(
        key.WithKeys("?"),
        key.WithHelp("?", "help"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
}
```

### Matching Keys in Update

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, m.keys.Up):
            m.cursor--
        case key.Matches(msg, m.keys.Down):
            m.cursor++
        case key.Matches(msg, m.keys.Quit):
            return m, tea.Quit
        }
    }
    return m, nil
}
```

### Context-Sensitive Keys

```go
// Disable/enable bindings based on state
func (m *Model) updateKeyStates() {
    m.keys.Up.SetEnabled(m.mode == ModeNormal)
    m.keys.Down.SetEnabled(m.mode == ModeNormal)
}
```

### Help Integration

```go
import "github.com/charmbracelet/bubbles/help"

// Implement help.KeyMap interface
func (k KeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down, k.Enter},
        {k.Back, k.Help, k.Quit},
    }
}

// In model
type model struct {
    keys KeyMap
    help help.Model
}

func (m model) View() string {
    content := m.renderContent()
    helpView := m.help.View(m.keys)
    return lipgloss.JoinVertical(lipgloss.Left, content, helpView)
}
```

## Common Components (Bubbles)

### Spinner

```go
import "github.com/charmbracelet/bubbles/spinner"

s := spinner.New()
s.Spinner = spinner.Dot  // Dot, Line, MiniDot, Jump, Pulse, etc.
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

// Must tick in Init
func (m model) Init() tea.Cmd {
    return m.spinner.Tick
}

// Update spinner
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg, ok := msg.(spinner.TickMsg); ok {
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}
```

### Text Input

```go
import "github.com/charmbracelet/bubbles/textinput"

ti := textinput.New()
ti.Placeholder = "Enter text..."
ti.Focus()
ti.CharLimit = 156
ti.Width = 30

// Update in Update()
m.input, cmd = m.input.Update(msg)

// Get value
value := m.input.Value()
```

### List

```go
import "github.com/charmbracelet/bubbles/list"

// Implement list.Item interface
type item struct {
    title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Create list
items := []list.Item{
    item{title: "Item 1", desc: "Description"},
}

l := list.New(items, list.NewDefaultDelegate(), width, height)
l.Title = "My List"
l.SetFilteringEnabled(true)
```

### Viewport (Scrollable)

```go
import "github.com/charmbracelet/bubbles/viewport"

vp := viewport.New(80, 20)
vp.SetContent(longContent)

// Handles scroll keys automatically (j/k, arrows, pgup/pgdn)
m.viewport, cmd = m.viewport.Update(msg)
```

### Progress Bar

```go
import "github.com/charmbracelet/bubbles/progress"

p := progress.New(progress.WithDefaultGradient())

// In View
progressView := p.ViewAs(0.5)  // 50% complete
```

## Dialogs and Modals

### Modal Overlay Pattern

```go
type App struct {
    mainContent tea.Model
    modal       tea.Model
    showModal   bool
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ShowModalMsg:
        a.showModal = true
        a.modal = NewConfirmDialog(msg.Title, msg.Message)
        return a, a.modal.Init()
        
    case HideModalMsg:
        a.showModal = false
        a.modal = nil
        return a, nil
    }
    
    // Route to modal if showing, else main content
    if a.showModal {
        var cmd tea.Cmd
        a.modal, cmd = a.modal.Update(msg)
        return a, cmd
    }
    
    var cmd tea.Cmd
    a.mainContent, cmd = a.mainContent.Update(msg)
    return a, cmd
}

func (a *App) View() string {
    main := a.mainContent.View()
    if a.showModal {
        // Center modal over main content
        modal := a.modal.View()
        return lipgloss.Place(
            a.width, a.height,
            lipgloss.Center, lipgloss.Center,
            modal,
        )
    }
    return main
}
```

### Confirm Dialog Example

```go
type ConfirmDialog struct {
    title    string
    message  string
    focused  int  // 0=No, 1=Yes
}

func (d *ConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg, ok := msg.(tea.KeyMsg); ok {
        switch msg.String() {
        case "left", "h":
            d.focused = 0
        case "right", "l":
            d.focused = 1
        case "enter":
            return d, func() tea.Msg {
                return ConfirmResultMsg{Confirmed: d.focused == 1}
            }
        case "esc":
            return d, func() tea.Msg { return HideModalMsg{} }
        }
    }
    return d, nil
}

func (d *ConfirmDialog) View() string {
    var noBtn, yesBtn string
    if d.focused == 0 {
        noBtn = focusedStyle.Render("[ No ]")
        yesBtn = normalStyle.Render("[ Yes ]")
    } else {
        noBtn = normalStyle.Render("[ No ]")
        yesBtn = focusedStyle.Render("[ Yes ]")
    }
    
    buttons := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "  ", yesBtn)
    content := lipgloss.JoinVertical(lipgloss.Center,
        d.title, "", d.message, "", buttons)
    
    return dialogStyle.Render(content)
}
```

## Styling (Lip Gloss)

### Basic Styling

```go
import "github.com/charmbracelet/lipgloss"

var style = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    Padding(1, 2).
    Margin(1, 0)

output := style.Render("Hello!")
```

### Colors

```go
// ANSI 256
lipgloss.Color("86")

// True color (hex)
lipgloss.Color("#FF5733")

// Adaptive (light/dark terminals)
lipgloss.AdaptiveColor{Light: "236", Dark: "248"}
```

### Borders

```go
var boxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("63")).
    Padding(1, 2)

// Border types: NormalBorder, RoundedBorder, BlockBorder,
// DoubleBorder, ThickBorder
```

### Layout

```go
// Horizontal join
row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
row := lipgloss.JoinHorizontal(lipgloss.Center, items...)

// Vertical join
col := lipgloss.JoinVertical(lipgloss.Left, top, bottom)
col := lipgloss.JoinVertical(lipgloss.Center, items...)

// Place content in space
centered := lipgloss.Place(width, height, 
    lipgloss.Center, lipgloss.Center, 
    content)
```

### Dimensions

```go
// Set size
style := lipgloss.NewStyle().
    Width(40).
    Height(10).
    MaxWidth(60)

// Measure rendered content
w := lipgloss.Width(rendered)
h := lipgloss.Height(rendered)
```

### Responsive Layout

```go
func (m model) View() string {
    if m.width < 80 {
        // Narrow: stack vertically
        return lipgloss.JoinVertical(lipgloss.Left,
            m.sidebar.View(),
            m.content.View(),
        )
    }
    // Wide: side by side
    return lipgloss.JoinHorizontal(lipgloss.Top,
        sidebarStyle.Render(m.sidebar.View()),
        contentStyle.Render(m.content.View()),
    )
}
```

## External Messages (from outside the program)

```go
p := tea.NewProgram(model{})

// Run in goroutine
go p.Run()

// Send messages from outside
p.Send(ExternalUpdateMsg{data})

// Useful for Pulumi event streaming:
go func() {
    for event := range pulumiEvents {
        p.Send(PulumiEventMsg{event})
    }
}()
```

## References

- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)
- [pkg.go.dev/github.com/charmbracelet/bubbletea](https://pkg.go.dev/github.com/charmbracelet/bubbletea)
