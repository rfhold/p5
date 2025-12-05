package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	_ "github.com/rfhold/p5/internal/plugins/builtins" // Register builtin plugins
)

// Package-level variables for CLI argument parsing.
// These are required by the flag package and are transferred to AppContext at startup.
var argWorkDir string
var argStackName string
var argDebug bool

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.MouseMsg:
		return m.handleMouseEvent(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	default:
		return m.handleMessage(msg)
	}
}

func main() {
	flag.StringVar(&argWorkDir, "C", "", "Run as if p5 was started in `path`")
	flag.StringVar(&argWorkDir, "cwd", "", "Run as if p5 was started in `path`")
	flag.StringVar(&argStackName, "s", "", "Select the Pulumi `stack` to use")
	flag.StringVar(&argStackName, "stack", "", "Select the Pulumi `stack` to use")
	flag.BoolVar(&argDebug, "debug", false, "Enable debug logging")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: p5 [flags] [command]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  up        Start with up preview\n")
		fmt.Fprintf(os.Stderr, "  refresh   Start with refresh preview\n")
		fmt.Fprintf(os.Stderr, "  destroy   Start with destroy preview\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Disable logging by default, enable with -debug flag
	if !argDebug {
		log.SetOutput(io.Discard)
	}

	// Get current working directory (where app was launched from)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Build AppContext from CLI arguments
	ctx := AppContext{
		Cwd:       cwd,
		StackName: argStackName,
		StartView: "stack",
	}

	// Get command from positional argument
	args := flag.Args()
	if len(args) > 0 {
		ctx.StartView = args[0]
	}

	// Default to current directory if not specified
	if argWorkDir == "" {
		ctx.WorkDir = cwd
	} else {
		ctx.WorkDir = argWorkDir
	}

	// Create production dependencies
	deps, err := NewProductionDependencies(ctx.WorkDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing dependencies: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(ctx, deps), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
