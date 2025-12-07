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

	h.WaitFor("RandomId", 30*time.Second)

	// Move to RandomId resource
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Target the RandomId - should show T:1 in status bar
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("targeted")

	// Also mark same resource for replace - should show T:1 R:1
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("target_and_replace")

	// Clear the flags with 'c' (clear current) then 'C' (clear all)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("cleared_current")

	h.Quit(5 * time.Second)
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

	h.WaitFor("RandomId", 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("visual_mode_entered")

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("visual_selection_extended")

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("excluded_selected")

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("cleared_one_flag")

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("cleared_all_flags")

	h.Quit(5 * time.Second)
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

	h.WaitFor("RandomId", 30*time.Second)

	// Move to RandomId
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Exclude the RandomId from destroy
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	time.Sleep(100 * time.Millisecond)
	h.Snapshot("excluded_randomid")

	// Start destroy preview
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	// Wait for destroy preview to complete - look for resources to appear
	// The preview should show -3 deletes (excluding the Stack which has [E] flag)
	h.WaitFor("-3", 30*time.Second)
	time.Sleep(200 * time.Millisecond)
	h.Snapshot("destroy_preview_with_exclusion")

	h.Quit(5 * time.Second)
}
