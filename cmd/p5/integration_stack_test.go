//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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

	h.WaitFor("RandomId", 30*time.Second)

	// Navigate down to RandomId (sequence order: Stack > RandomId, RandomString, then Provider)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	// Details panel shows "Properties" header and "byteLength" for RandomId
	h.WaitAndSnapshot("byteLength", "details_panel", 5*time.Second)

	h.Quit(5 * time.Second)
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

	h.WaitFor("RandomId", 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	// History view shows entries - wait for actual history entry to appear
	// Wait for "succeeded" which only appears when history items are loaded
	h.WaitAndSnapshot("succeeded", "history_view", 10*time.Second)

	h.Quit(5 * time.Second)
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

	h.WaitFor("RandomId", 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	// Confirmation modal shows "Execute Destroy" title
	h.WaitAndSnapshot("Execute Destroy", "destroy_confirmation", 5*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	time.Sleep(200 * time.Millisecond)
	h.Snapshot("cancelled")

	h.Quit(5 * time.Second)
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

	h.WaitFor("RandomId", 30*time.Second)

	// Navigate down to RandomId (sequence order: Stack > RandomId, RandomString, then Provider)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	h.WaitAndSnapshot("delete", "remove_confirmation", 5*time.Second)

	h.Quit(5 * time.Second)
}
