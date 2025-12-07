//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFlags_TargetReplace(t *testing.T) {
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

	// Move to RandomId resource
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Target the RandomId - should show T:1 in status bar
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	h.WaitFor("T:1", 2*time.Second)

	// Also mark same resource for replace - should show T:1 R:1
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	h.WaitFor("R:1", 2*time.Second)

	// Clear the flags with 'c' (clear current)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	time.Sleep(100 * time.Millisecond)

	h.Snapshot("cleared_current")
}

func TestFlags_VisualExcludeClearAll(t *testing.T) {
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

	// Move to RandomId
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Enter visual mode
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	h.WaitFor("VISUAL", 2*time.Second)

	// Extend selection
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Exclude selected resources
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	h.WaitFor("E:2", 2*time.Second)

	// Clear one flag
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	time.Sleep(100 * time.Millisecond)

	// Clear all flags
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	time.Sleep(100 * time.Millisecond)

	h.Snapshot("cleared_all_flags")
}

func TestFlags_ExcludeFromDestroy(t *testing.T) {
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

	// Move to RandomId
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Exclude the RandomId from destroy
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	h.WaitFor("E:1", 2*time.Second)

	// Start destroy preview
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	// Wait for destroy preview to complete - footer shows "ctrl+u execute" when done
	// The preview should show -3 deletes (excluding the resource with [E] flag)
	h.WaitForAll([]string{"-3", "ctrl+u execute"}, 30*time.Second)
	h.FinalSnapshot("destroy_preview_with_exclusion")
}
