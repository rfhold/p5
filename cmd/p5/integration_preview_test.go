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
	// Also wait for footer keys to confirm settled state
	h.WaitForAll([]string{"done", "ctrl+u execute"}, 30*time.Second)
	h.FinalSnapshot("preview_up_done")
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
	h.WaitForAll([]string{"done", "ctrl+u execute"}, 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Wait for execution to complete - "done" reappears after execution finishes.
	// Also verify resources show "created" status and "esc cancel" for execute view.
	h.WaitForAll([]string{
		"created",
		"done",
		"esc cancel",
	}, 30*time.Second)
	h.FinalSnapshot("execute_up_done")
}

func TestPreview_Destroy(t *testing.T) {
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
	h.WaitForAll([]string{"done", "ctrl+u execute"}, 30*time.Second)
	h.FinalSnapshot("preview_destroy_done")
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
	h.WaitForAll([]string{"done", "ctrl+u execute"}, 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	// Wait for execution to complete - "done" reappears after execution finishes.
	// Also verify resources show "deleted" status.
	h.WaitForAll([]string{
		"deleted",
		"done",
		"esc cancel",
	}, 30*time.Second)
	h.FinalSnapshot("execute_destroy_done")
}

func TestPreview_Refresh(t *testing.T) {
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
	h.WaitForAll([]string{"done", "ctrl+u execute"}, 30*time.Second)
	h.FinalSnapshot("preview_refresh_done")
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
	h.WaitForAll([]string{"done", "ctrl+u execute"}, 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlR})

	// Wait for execution to complete - "done" reappears after execution finishes.
	// Also verify resources show "refreshed" status.
	h.WaitForAll([]string{
		"refreshed",
		"done",
		"esc cancel",
	}, 30*time.Second)
	h.FinalSnapshot("execute_refresh_done")
}
