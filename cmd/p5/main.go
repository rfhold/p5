package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	_ "github.com/rfhold/p5/internal/plugins/builtins" // Register builtin plugins
	"github.com/rfhold/p5/internal/telemetry"
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

	// Initialize telemetry (configured via OTEL_* environment variables)
	// Debug flag enables local stderr logging when OTEL endpoint is not configured
	tel, err := telemetry.Setup(context.Background(), telemetry.Options{Debug: argDebug})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to setup telemetry: %v\n", err)
		tel = telemetry.NewNoop()
	}
	defer tel.Shutdown(context.Background())

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
	deps := NewProductionDependencies(ctx.WorkDir, tel.Logger)

	// Create application-level context with cancellation for graceful shutdown.
	// This context is passed through to all async operations, enabling them to
	// be cancelled when the application exits (via signal or user quit).
	appCtx, appCancel := context.WithCancel(context.Background())

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		appCancel()
	}()

	p := tea.NewProgram(initialModel(appCtx, ctx, deps), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	appCancel() // Cancel context before potential exit
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
