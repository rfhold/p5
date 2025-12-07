//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelp_ShowAndClose(t *testing.T) {
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

	// Open help with ?
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	h.WaitFor("Keyboard Shortcuts", 5*time.Second)

	// Close help with ? again
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	time.Sleep(200 * time.Millisecond)

	h.Snapshot("help_closed")
}
