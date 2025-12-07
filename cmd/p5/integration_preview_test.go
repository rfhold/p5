//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPreview_Up(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}

	m := te.CreateModel("up")
	h := newTestHarness(t, m)

	// Wait for preview to complete - "done" appears in header only when fully complete.
	h.WaitFor("done", 15*time.Second)
}

func TestPreview_UpAndExecute(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}

	m := te.CreateModel("up")
	h := newTestHarness(t, m)

	// Wait for preview to complete - "done" appears in header only when fully complete.
	h.WaitFor("done", 15*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Wait for execution to complete - "done" reappears after execution finishes.
	// Also verify resources show "created" status.
	h.WaitForAll([]string{
		"created",
		"done",
	}, 15*time.Second)
}

func TestPreview_DestroyAndExecute(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}

	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy stack: %v", err)
	}

	m := te.CreateModel("destroy")
	h := newTestHarness(t, m)

	// Wait for preview to complete - "done" appears in header only when fully complete.
	h.WaitFor("done", 15*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	// Wait for execution to complete - "done" reappears after execution finishes.
	// Also verify resources show "deleted" status.
	h.WaitForAll([]string{
		"deleted",
		"done",
	}, 15*time.Second)
}

func TestPreview_RefreshAndExecute(t *testing.T) {
	t.Parallel()

	te := SetupTestEnv(t, "simple")
	ctx := context.Background()

	if err := te.CreateStack(ctx); err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}

	if err := te.DeployStack(ctx); err != nil {
		t.Fatalf("failed to deploy stack: %v", err)
	}

	// Delay to ensure pulumi releases any locks from the deploy
	time.Sleep(2 * time.Second)

	m := te.CreateModel("refresh")
	h := newTestHarness(t, m)

	// Wait for preview to complete - "done" appears in header only when fully complete.
	h.WaitFor("done", 15*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlR})

	// Wait for execution to complete - "done" reappears after execution finishes.
	// Also verify resources show "refreshed" status.
	h.WaitForAll([]string{
		"refreshed",
		"done",
	}, 15*time.Second)
}
