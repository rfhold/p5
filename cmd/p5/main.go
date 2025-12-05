package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	_ "github.com/rfhold/p5/internal/plugins/builtins" // Register builtin plugins
)

// Package-level variables for CLI arguments
var workDir string
var stackName string
var startView string

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
	flag.StringVar(&workDir, "C", "", "Run as if p5 was started in `path`")
	flag.StringVar(&workDir, "cwd", "", "Run as if p5 was started in `path`")
	flag.StringVar(&stackName, "s", "", "Select the Pulumi `stack` to use")
	flag.StringVar(&stackName, "stack", "", "Select the Pulumi `stack` to use")
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

	// Get command from positional argument
	args := flag.Args()
	if len(args) > 0 {
		startView = args[0]
	} else {
		startView = "stack"
	}

	// Default to current directory if not specified
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
