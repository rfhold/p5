//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestInit_NeedsWorkspaceSelection(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "nested-workspace")

	deps := &Dependencies{
		StackOperator:    pulumi.NewStackOperator(),
		StackReader:      pulumi.NewStackReader(),
		WorkspaceReader:  pulumi.NewWorkspaceReader(),
		StackInitializer: pulumi.NewStackInitializer(),
		ResourceImporter: pulumi.NewResourceImporter(),
		PluginProvider:   &plugins.FakePluginProvider{},
		Env:              te.Env,
		Logger:           discardLogger(),
	}

	appCtx := AppContext{
		Cwd:       te.WorkDir,
		WorkDir:   te.WorkDir,
		StackName: "",
		StartView: "stack",
	}

	m := initialModel(context.Background(), appCtx, deps)
	h := newTestHarness(t, m)

	h.WaitAndSnapshot("Select Workspace", "workspace_selector_shown", 10*time.Second)
}

func TestInit_MultipleStacksAutoSelectsCurrent(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "multi-stack")
	ctx := context.Background()

	// Create dev stack first
	te.StackName = "dev"
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create dev stack: %v", err)
	}

	// Create staging stack - this becomes the "current" stack in Pulumi
	te.StackName = "staging"
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create staging stack: %v", err)
	}

	deps := &Dependencies{
		StackOperator:    pulumi.NewStackOperator(),
		StackReader:      pulumi.NewStackReader(),
		WorkspaceReader:  pulumi.NewWorkspaceReader(),
		StackInitializer: pulumi.NewStackInitializer(),
		ResourceImporter: pulumi.NewResourceImporter(),
		PluginProvider:   &plugins.FakePluginProvider{},
		Env:              te.Env,
		Logger:           discardLogger(),
	}

	appCtx := AppContext{
		Cwd:       te.WorkDir,
		WorkDir:   te.WorkDir,
		StackName: "",
		StartView: "stack",
	}

	m := initialModel(context.Background(), appCtx, deps)
	h := newTestHarness(t, m)

	// App should auto-select the current stack (staging) and show settled stack view
	// Wait for both stack name in header AND footer showing stack view keys
	h.WaitForAll([]string{
		"Stack: staging",
		"u up",
	}, 15*time.Second)
	h.FinalSnapshot("auto_selected_current_stack")
}

func TestInit_CreateNewStack(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")

	deps := &Dependencies{
		StackOperator:    pulumi.NewStackOperator(),
		StackReader:      pulumi.NewStackReader(),
		WorkspaceReader:  pulumi.NewWorkspaceReader(),
		StackInitializer: pulumi.NewStackInitializer(),
		ResourceImporter: pulumi.NewResourceImporter(),
		PluginProvider:   &plugins.FakePluginProvider{},
		Env:              te.Env,
		Logger:           discardLogger(),
	}

	appCtx := AppContext{
		Cwd:       te.WorkDir,
		WorkDir:   te.WorkDir,
		StackName: "",
		StartView: "stack",
	}

	m := initialModel(context.Background(), appCtx, deps)
	h := newTestHarness(t, m)

	// Wait for modal AND backend info to be loaded
	h.WaitFor("Backend:", 10*time.Second)

	for _, r := range "teststack" {
		h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		time.Sleep(20 * time.Millisecond)
	}

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})
	h.WaitFor("Select secrets provider", 5*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})
	h.WaitFor("Enter passphrase", 5*time.Second)

	for _, r := range "testpassphrase" {
		h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		time.Sleep(20 * time.Millisecond)
	}

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for fully settled state: header shows stack name, toast shown, footer shows keys
	h.WaitForAll([]string{
		"Stack: teststack",
		"Created stack 'teststack'",
		"u up",
	}, 30*time.Second)
	h.FinalSnapshot("stack_created")
}

func TestInit_SelectExistingStack(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "multi-stack")
	ctx := context.Background()

	// Create and deploy dev stack
	te.StackName = "dev"
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create dev stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy dev stack: %v", err)
	}

	// Create staging stack (no deployment needed)
	te.StackName = "staging"
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create staging stack: %v", err)
	}

	// Start with dev stack which has resources
	te.StackName = "dev"
	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for stack view to be fully loaded (footer shows stack view keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	// Open stack selector
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Wait for stacks to be loaded (not just the modal title, but actual stack names)
	h.WaitForAll([]string{"Select Stack", "dev", "staging"}, 10*time.Second)

	// Navigate to staging (down from dev) - use 'j' key which also works for down
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(100 * time.Millisecond)

	// Select staging
	h.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for staging stack to load with settled view (stack name + footer)
	h.WaitForAll([]string{
		"Stack: staging",
		"u up",
	}, 30*time.Second)
	h.FinalSnapshot("switched_to_staging")
}

func TestInit_WorkspaceSelectionAndNavigate(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "nested-workspace")
	ctx := context.Background()

	projectDir := filepath.Join(te.WorkDir, "project")

	te.StackName = "dev"
	originalWorkDir := te.WorkDir
	te.WorkDir = projectDir
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create dev stack: %v", err)
	}
	te.WorkDir = originalWorkDir

	deps := &Dependencies{
		StackOperator:    pulumi.NewStackOperator(),
		StackReader:      pulumi.NewStackReader(),
		WorkspaceReader:  pulumi.NewWorkspaceReader(),
		StackInitializer: pulumi.NewStackInitializer(),
		ResourceImporter: pulumi.NewResourceImporter(),
		PluginProvider:   &plugins.FakePluginProvider{},
		Env:              te.Env,
		Logger:           discardLogger(),
	}

	appCtx := AppContext{
		Cwd:       te.WorkDir,
		WorkDir:   te.WorkDir,
		StackName: "",
		StartView: "stack",
	}

	m := initialModel(context.Background(), appCtx, deps)
	h := newTestHarness(t, m)

	h.WaitFor("Select Workspace", 10*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// After selecting the workspace, wait for settled stack view with program name + footer
	h.WaitForAll([]string{
		"Program: nested-project",
		"u up",
	}, 30*time.Second)
	h.FinalSnapshot("after_workspace_selection")
}
