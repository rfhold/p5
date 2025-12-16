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

func TestStack_DiscreteSelect(t *testing.T) {
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

	// Press space to discretely select RandomId
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Move to RandomString
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Press space to discretely select RandomString
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Apply Target flag to verify both items are selected - should show T:2
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:2", 2*time.Second)

	h.FinalSnapshot("discrete_select")
}

func TestStack_DiscreteSelectWithVisualMode(t *testing.T) {
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

	// Discretely select the stack (first item)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Move to RandomString
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Enter visual mode at RandomString
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	h.WaitFor("VISUAL", 2*time.Second)

	// Extend visual selection to provider
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Apply Target flag - should apply to union: 1 discrete (stack) + 2 visual (RandomString, Provider) = T:3
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:3", 2*time.Second)

	h.FinalSnapshot("discrete_and_visual")
}

func TestStack_DiscreteSelectFlags(t *testing.T) {
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

	// Discretely select RandomId
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Move to RandomString and discretely select it
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Apply Target flag - should apply to both discrete selections
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:2", 2*time.Second)

	h.FinalSnapshot("discrete_select_flags")
}

func TestStack_DiscreteSelectEscapeClear(t *testing.T) {
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

	// Move to RandomId and discretely select it
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Press escape - should clear discrete selections
	h.Send(tea.KeyMsg{Type: tea.KeyEscape})
	time.Sleep(100 * time.Millisecond)

	// Apply Target flag - should only apply to cursor item (T:1), not T:2
	// This verifies the discrete selection was cleared
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:1", 2*time.Second)

	h.FinalSnapshot("discrete_select_cleared")
}

func TestStack_DiscreteSelectVisualFirst(t *testing.T) {
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

	// Discretely select first item (Stack)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Move down and enter visual mode
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	h.WaitFor("VISUAL", 2*time.Second)

	// First escape exits visual mode
	h.Send(tea.KeyMsg{Type: tea.KeyEscape})
	time.Sleep(100 * time.Millisecond)

	// Apply Target flag - should apply to discrete selection only (T:1 for Stack)
	// proving discrete selection persisted after visual mode exit
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:1", 2*time.Second)

	h.FinalSnapshot("after_first_escape")
}

func TestStack_DiscreteSelectInVisualRange(t *testing.T) {
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

	// Extend to RandomString
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Press space to discretely select all items in visual range
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(100 * time.Millisecond)

	// Exit visual mode
	h.Send(tea.KeyMsg{Type: tea.KeyEscape})
	time.Sleep(100 * time.Millisecond)

	// Apply Target flag - should apply to both discretely selected items (T:2)
	// This verifies that space in visual mode converted the range to discrete selections
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:2", 2*time.Second)

	h.FinalSnapshot("visual_range_to_discrete")
}
