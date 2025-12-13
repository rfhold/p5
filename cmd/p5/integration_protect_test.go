//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// sendString sends each character in a string as key presses
func sendString(h *testHarness, s string) {
	for _, c := range s {
		h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{c}})
	}
}

// filterToResource uses the filter feature to navigate to a specific resource by name
// This avoids flakiness from resource order changes between Pulumi executions
func filterToResource(h *testHarness, resourceName string) {
	// Press '/' to activate filter
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	time.Sleep(50 * time.Millisecond)

	// Type the resource name to filter
	sendString(h, resourceName)
	time.Sleep(50 * time.Millisecond)

	// Press Escape to exit filter input mode (keeps filter text applied)
	h.Send(tea.KeyMsg{Type: tea.KeyEscape})
	time.Sleep(100 * time.Millisecond)

	// Press 'j' to move to the first matching resource (since filter matches show below root)
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	// Clear the filter with Escape so commands work normally
	h.Send(tea.KeyMsg{Type: tea.KeyEscape})
	time.Sleep(100 * time.Millisecond)
}

func TestProtect_ProtectResource(t *testing.T) {
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

	// Wait for settled stack view (resources loaded)
	h.WaitForAll([]string{"base-id", "u up"}, 30*time.Second)

	// Use filter to navigate to base-id resource reliably
	filterToResource(h, "base-id")

	// Protect the resource with 'P' - should execute immediately without confirmation
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})

	// Wait for protected badge to appear on the resource
	h.WaitFor("Protected", 10*time.Second)

	// Take a snapshot showing the protected resource
	h.FinalSnapshot("resource_protected")
}

func TestProtect_UnprotectResourceFlow(t *testing.T) {
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

	// Wait for settled stack view (resources loaded)
	h.WaitForAll([]string{"base-id", "u up"}, 30*time.Second)

	// Use filter to navigate to base-id resource reliably
	filterToResource(h, "base-id")

	// First protect the resource
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	h.WaitFor("Protected", 10*time.Second)

	// After protect completes, the list reloads - use filter again to find resource
	time.Sleep(500 * time.Millisecond)
	filterToResource(h, "base-id")

	// Ensure we're on the protected resource and ready for input
	time.Sleep(200 * time.Millisecond)

	// Now try to unprotect - should show confirmation modal
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})

	// Wait for the unprotect confirmation modal
	h.WaitFor("Unprotect Resource", 5*time.Second)

	// Confirm the unprotect with 'y'
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Wait for unprotected toast message
	h.WaitFor("Unprotected", 10*time.Second)

	h.FinalSnapshot("resource_unprotected")
}

func TestProtect_UnprotectModalCancel(t *testing.T) {
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
	h.WaitForAll([]string{"base-id", "u up"}, 30*time.Second)

	// Use filter to navigate to base-id resource reliably
	filterToResource(h, "base-id")

	// Protect the resource first
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	h.WaitFor("Protected", 10*time.Second)

	// After protect completes, the list reloads - use filter again
	time.Sleep(500 * time.Millisecond)
	filterToResource(h, "base-id")

	// Ensure we're on the protected resource and ready for input
	time.Sleep(200 * time.Millisecond)

	// Try to unprotect - should show confirmation modal
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	h.WaitFor("Unprotect Resource", 5*time.Second)

	// Cancel with 'n' - resource should stay protected
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Wait for modal to close and verify Protected badge is still visible
	time.Sleep(200 * time.Millisecond)
	h.WaitFor("Protected", 2*time.Second)

	h.FinalSnapshot("unprotect_cancelled")
}

func TestProtect_CannotProtectStackRoot(t *testing.T) {
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
	h.WaitForAll([]string{"base-id", "u up"}, 30*time.Second)

	// Stay on the root stack resource (first item, pulumi:pulumi:Stack)
	// Try to protect - should do nothing since root stack cannot be protected
	h.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})

	// Wait a moment and verify no Protected badge appears
	time.Sleep(500 * time.Millisecond)

	// The stack root should not show Protected badge
	h.FinalSnapshot("stack_root_not_protected")
}
