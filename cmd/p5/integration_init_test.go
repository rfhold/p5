//go:build integration

package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
)

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

	h.Quit(5 * time.Second)
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
	}

	appCtx := AppContext{
		Cwd:       te.WorkDir,
		WorkDir:   te.WorkDir,
		StackName: "",
		StartView: "stack",
	}

	m := initialModel(context.Background(), appCtx, deps)
	h := newTestHarness(t, m)

	// App should auto-select the current stack (staging) and load it
	h.WaitAndSnapshot("staging", "auto_selected_current_stack", 15*time.Second)

	h.Quit(5 * time.Second)
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
	}

	appCtx := AppContext{
		Cwd:       te.WorkDir,
		WorkDir:   te.WorkDir,
		StackName: "",
		StartView: "stack",
	}

	m := initialModel(context.Background(), appCtx, deps)
	h := newTestHarness(t, m)

	// Wait for modal AND backend info to be loaded (to avoid race condition)
	h.WaitFor("Backend:", 10*time.Second)
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("stack_init_modal_shown")

	for _, r := range "teststack" {
		h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		time.Sleep(20 * time.Millisecond)
	}

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	h.WaitFor("Select secrets provider", 5*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	h.WaitFor("Enter passphrase", 5*time.Second)

	for _, r := range "testpassphrase" {
		h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		time.Sleep(20 * time.Millisecond)
	}

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})

	h.WaitAndSnapshot("teststack", "stack_created", 30*time.Second)

	h.Quit(5 * time.Second)
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

	h.WaitFor("RandomId", 30*time.Second)
	h.Snapshot("initial_stack_view")

	// Open stack selector
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	h.WaitFor("Select Stack", 5*time.Second)
	h.Snapshot("stack_selector_open")

	// Navigate to staging (2 down from dev)
	h.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)

	// Select staging
	h.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for staging stack to load (it has 0 resources)
	h.WaitAndSnapshot("staging", "switched_to_staging", 30*time.Second)

	h.Quit(5 * time.Second)
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
	h.Snapshot("workspace_selector_initial")

	h.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// After selecting the workspace, the app loads the project (named "nested-project")
	h.WaitAndSnapshot("nested-project", "after_workspace_selection", 30*time.Second)

	h.Quit(5 * time.Second)
}
