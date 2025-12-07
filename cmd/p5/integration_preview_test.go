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

	// Wait for preview to complete
	h.WaitFor("RandomId", 30*time.Second)
	h.FinalSnapshot("preview_creates")
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

	// Wait for preview to complete
	h.WaitFor("RandomId", 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Wait for all resources to be created - provider is typically last
	h.WaitFor("default_4_16_2 created", 30*time.Second)
	h.FinalSnapshot("execute_complete")
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

	// Wait for preview to complete
	h.WaitFor("RandomId", 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	// Wait for all resources to be deleted - provider is typically last
	h.WaitFor("default_4_16_2 deleted", 30*time.Second)
	h.FinalSnapshot("destroy_complete")
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

	m := te.CreateModel("refresh")
	h := newTestHarness(t, m)

	// Wait for preview to complete
	h.WaitFor("RandomId", 30*time.Second)

	h.Send(tea.KeyMsg{Type: tea.KeyCtrlR})

	// Wait for all resources to be refreshed - provider is typically last
	h.WaitFor("default_4_16_2 refreshed", 30*time.Second)
	h.FinalSnapshot("refresh_complete")
}
