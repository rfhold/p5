//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStack_View(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "multi")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resources loaded + footer keys visible)
	h.WaitForAll([]string{"RandomId", "RandomString", "u up"}, 30*time.Second)
	h.FinalSnapshot("stack_view")
}

func TestStack_VisualMode(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "multi")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	// Move to RandomId
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Enter visual mode
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	h.WaitFor("VISUAL", 2*time.Second)

	// Extend selection down
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	h.Snapshot("visual_mode")
}

func TestStack_StackSelector(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "multi-stack")
	ctx := context.Background()

	// Create dev stack
	te.StackName = "dev"
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create dev stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy dev stack: %v", err)
	}

	// Create staging stack
	te.StackName = "staging"
	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create staging stack: %v", err)
	}

	// Start with dev stack
	te.StackName = "dev"
	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for stack view to be fully loaded
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	// Open stack selector with 's'
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	// Wait for stack selector to show both stacks
	h.WaitForAll([]string{"Select Stack", "dev", "staging"}, 10*time.Second)
	h.FinalSnapshot("stack_selector_open")
}

func TestStack_ShowResourceDetails(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "multi")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resource + footer keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	// Navigate down to RandomId (sequence order: Stack > RandomId, RandomString, then Provider)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	// Details panel shows "Properties" header and "byteLength" for RandomId
	h.WaitAndSnapshot("byteLength", "details_panel", 5*time.Second)
}

func TestStack_NavigateToHistory(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resource + footer keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	// History view shows entries - wait for actual history entry to appear
	// Wait for "succeeded" which only appears when history items are loaded
	h.WaitAndSnapshot("succeeded", "history_view", 10*time.Second)
}

func TestHistory_ViewDetails(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resource + footer keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	// Navigate to history
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	h.WaitFor("succeeded", 10*time.Second)

	// Open details panel for the history entry
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	// History detail shows "Update #" and details about the operation
	h.WaitAndSnapshot("Update #", "history_details_panel", 5*time.Second)
}

func TestStack_DestroyConfirmationModal(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resource + footer keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	// Confirmation modal shows "Execute Destroy" title
	h.WaitAndSnapshot("Execute Destroy", "destroy_confirmation_modal", 5*time.Second)
}

func TestStack_DirectDestroyConfirmation(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resource + footer keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	// Confirmation modal shows "Execute Destroy" title
	h.WaitFor("Execute Destroy", 5*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Verify modal is closed by checking we're back to stack view
	h.WaitFor("u up", 2*time.Second)
}

func TestStack_RemoveFromState(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}
	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy: %v", err)
	}

	m := te.CreateModel("stack")
	h := newTestHarness(t, m)

	// Wait for settled stack view (resource + footer keys)
	h.WaitForAll([]string{"RandomId", "u up"}, 30*time.Second)

	// Navigate down to RandomId (sequence order: Stack > RandomId, RandomString, then Provider)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	h.WaitAndSnapshot("delete", "remove_confirmation", 5*time.Second)
}
