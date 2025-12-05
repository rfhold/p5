# Bubble Tea Integration Testing Plan

This document outlines the strategy for adding integration tests to the p5 TUI application using the `teatest` package and related tooling.

## Overview

Our current unit test coverage focuses on pure logic functions and state transitions. Integration tests will validate complete user flows through the TUI, ensuring that key sequences, view rendering, and async operations work correctly together.

### Testing Pyramid for p5

| Layer | Tool | Speed | What It Tests |
|-------|------|-------|---------------|
| Unit | Direct model calls | Fast (ms) | Logic, state transitions, pure functions |
| Integration | teatest | Medium (100ms-1s) | Full message flow, key handling, view output |
| Golden File | teatest + RequireEqualOutput | Medium | View regression detection |
| E2E/Visual | VHS | Slow (seconds) | User flows, demo generation |

## Prerequisites

### 1. Install teatest Package

```bash
go get github.com/charmbracelet/x/exp/teatest@latest
```

> **Note**: teatest is in the experimental `x/exp` namespace. API may change.

### 2. Configure Color Profile for Reproducible Tests

Create or update `cmd/p5/main_test.go`:

```go
package main

import (
    "github.com/charmbracelet/lipgloss"
    "github.com/muesli/termenv"
)

func init() {
    // Force consistent color profile for reproducible tests across environments
    lipgloss.SetColorProfile(termenv.Ascii)
}
```

### 3. Configure Git for Golden Files

Add to `.gitattributes`:

```
*.golden -text
```

This prevents git from modifying line endings in golden files.

---

## Part 1: Test Infrastructure Setup

### 1.1 Create Integration Test File

Create `cmd/p5/integration_test.go` with build tag:

```go
//go:build integration

package main

import (
    "bytes"
    "context"
    "io"
    "testing"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/x/exp/teatest"
)

// Integration tests for complete user flows
```

### 1.2 Test Helper Functions

```go
// createTestModel creates a Model with fake dependencies for integration testing
func createTestModel(t *testing.T, opts ...testModelOption) Model {
    t.Helper()
    
    ctx := context.Background()
    deps := &Dependencies{
        StackOperator:   &FakeStackOperator{},
        StackReader:     &FakeStackReader{},
        WorkspaceReader: &FakeWorkspaceReader{},
        PluginManager:   &FakePluginManager{},
        Clipboard:       &FakeClipboard{},
    }
    
    appCtx := AppContext{
        WorkDir:   "/fake/workdir",
        StackName: "dev",
        StartView: "resources",
    }
    
    for _, opt := range opts {
        opt(deps, &appCtx)
    }
    
    return initialModel(ctx, appCtx, deps)
}

type testModelOption func(*Dependencies, *AppContext)

func withStackOperator(op StackOperator) testModelOption {
    return func(d *Dependencies, _ *AppContext) {
        d.StackOperator = op
    }
}

func withStartView(view string) testModelOption {
    return func(_ *Dependencies, ctx *AppContext) {
        ctx.StartView = view
    }
}

func withResources(resources []ResourceState) testModelOption {
    return func(d *Dependencies, _ *AppContext) {
        d.StackReader = &FakeStackReader{
            Resources: resources,
        }
    }
}
```

### 1.3 Event Collection Helper

```go
// waitForContent waits for specific content to appear in output
func waitForContent(t *testing.T, tm *teatest.TestModel, content string, timeout time.Duration) {
    t.Helper()
    teatest.WaitFor(t, tm.Output(),
        func(bts []byte) bool {
            return bytes.Contains(bts, []byte(content))
        },
        teatest.WithCheckInterval(50*time.Millisecond),
        teatest.WithDuration(timeout),
    )
}

// waitForAnyContent waits for any of the specified content strings
func waitForAnyContent(t *testing.T, tm *teatest.TestModel, contents []string, timeout time.Duration) {
    t.Helper()
    teatest.WaitFor(t, tm.Output(),
        func(bts []byte) bool {
            for _, content := range contents {
                if bytes.Contains(bts, []byte(content)) {
                    return true
                }
            }
            return false
        },
        teatest.WithCheckInterval(50*time.Millisecond),
        teatest.WithDuration(timeout),
    )
}
```

---

## Part 2: Core User Flow Tests

### 2.1 Initialization Flow Tests

Test the complete initialization sequence: workspace check -> plugin loading -> stack selection -> resource loading.

```go
func TestInitializationFlow_Success(t *testing.T) {
    m := createTestModel(t,
        withResources([]ResourceState{
            {URN: "urn:pulumi:dev::test::pkg:mod:Resource::myresource", Type: "pkg:mod:Resource"},
        }),
    )
    
    tm := teatest.NewTestModel(t, m,
        teatest.WithInitialTermSize(120, 40),
    )
    
    // Wait for resources to load and display
    waitForContent(t, tm, "myresource", 5*time.Second)
    
    // Verify we can quit cleanly
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestInitializationFlow_WorkspaceInvalid(t *testing.T) {
    // Test error handling when workspace is invalid
    deps := &Dependencies{
        WorkspaceReader: &FakeWorkspaceReader{
            IsValidWorkspaceError: errors.New("not a pulumi project"),
        },
    }
    // ... test error display
}

func TestInitializationFlow_PluginAuthFailure(t *testing.T) {
    // Test handling of plugin authentication failures
    deps := &Dependencies{
        PluginManager: &FakePluginManager{
            AuthResults: []PluginAuthResult{
                {PluginName: "aws", Success: false, Error: errors.New("credentials expired")},
            },
        },
    }
    // ... test error display and recovery options
}
```

### 2.2 Navigation Tests

Test keyboard navigation through the resource tree and list views.

```go
func TestNavigation_ResourceList(t *testing.T) {
    resources := []ResourceState{
        {URN: "urn:pulumi:dev::test::pkg:mod:A::first", Type: "pkg:mod:A"},
        {URN: "urn:pulumi:dev::test::pkg:mod:B::second", Type: "pkg:mod:B"},
        {URN: "urn:pulumi:dev::test::pkg:mod:C::third", Type: "pkg:mod:C"},
    }
    
    m := createTestModel(t, withResources(resources))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    // Wait for initial load
    waitForContent(t, tm, "first", 3*time.Second)
    
    // Navigate down
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
    
    // Navigate up
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
    
    // Toggle tree/list view
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
    
    // Quit
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}

func TestNavigation_FocusPanels(t *testing.T) {
    // Test Tab/Shift+Tab to move between panels
    m := createTestModel(t, withResources(testResources))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    waitForContent(t, tm, "resource", 3*time.Second)
    
    // Tab to details panel
    tm.Send(tea.KeyMsg{Type: tea.KeyTab})
    
    // Shift+Tab back to resource list
    tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}
```

### 2.3 Preview Operation Flow

Test the complete preview workflow: trigger -> progress -> results display.

```go
func TestPreviewFlow_Success(t *testing.T) {
    previewEvents := []PreviewEvent{
        {Step: &PreviewStep{URN: "urn:pulumi:dev::test::pkg:mod:Res::new", Op: OpCreate, New: &ResourceState{}}},
        {Step: &PreviewStep{URN: "urn:pulumi:dev::test::pkg:mod:Res::updated", Op: OpUpdate, Old: &ResourceState{}, New: &ResourceState{}}},
        {Done: true, Summary: &PreviewSummary{Create: 1, Update: 1}},
    }
    
    m := createTestModel(t,
        withStartView("up"),
        withStackOperator(&FakeStackOperator{PreviewEvents: previewEvents}),
    )
    
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    // Wait for preview to complete
    waitForContent(t, tm, "create", 5*time.Second)
    waitForContent(t, tm, "update", 5*time.Second)
    
    // Verify summary is shown
    waitForAnyContent(t, tm, []string{"1 to create", "1 to update"}, 2*time.Second)
    
    // Cancel without executing
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}

func TestPreviewFlow_WithErrors(t *testing.T) {
    previewEvents := []PreviewEvent{
        {Error: errors.New("resource validation failed")},
        {Done: true},
    }
    
    m := createTestModel(t,
        withStartView("up"),
        withStackOperator(&FakeStackOperator{PreviewEvents: previewEvents}),
    )
    
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    // Wait for error to display
    waitForContent(t, tm, "validation failed", 5*time.Second)
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}
```

### 2.4 Up/Destroy Operation Flow

Test executing operations after preview confirmation.

```go
func TestUpFlow_ConfirmAndExecute(t *testing.T) {
    previewEvents := []PreviewEvent{
        {Step: &PreviewStep{URN: "urn:...", Op: OpCreate}},
        {Done: true},
    }
    operationEvents := []OperationEvent{
        {Step: &OperationStep{URN: "urn:...", Op: OpCreate, Status: "creating"}},
        {Step: &OperationStep{URN: "urn:...", Op: OpCreate, Status: "created"}},
        {Done: true, Success: true},
    }
    
    m := createTestModel(t,
        withStartView("up"),
        withStackOperator(&FakeStackOperator{
            PreviewEvents:   previewEvents,
            OperationEvents: operationEvents,
        }),
    )
    
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    // Wait for preview
    waitForContent(t, tm, "create", 5*time.Second)
    
    // Confirm execution
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
    
    // Wait for operation to complete
    waitForContent(t, tm, "created", 10*time.Second)
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}

func TestDestroyFlow_RequiresConfirmation(t *testing.T) {
    // Test that destroy requires explicit confirmation
    // and shows appropriate warnings
}
```

### 2.5 Import Flow

Test the resource import modal and workflow.

```go
func TestImportFlow_OpenModalAndCancel(t *testing.T) {
    m := createTestModel(t, withResources(testResources))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    waitForContent(t, tm, "resource", 3*time.Second)
    
    // Open import modal
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
    
    // Verify modal is shown
    waitForContent(t, tm, "Import", 2*time.Second)
    
    // Cancel with Escape
    tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}

func TestImportFlow_SubmitImport(t *testing.T) {
    // Test typing import details and submitting
}
```

---

## Part 3: Modal and Dialog Tests

### 3.1 Confirm Modal Tests

```go
func TestConfirmModal_YesNoNavigation(t *testing.T) {
    // Test navigating between Yes/No options
    // Test Enter to confirm selection
    // Test Escape to cancel
}

func TestConfirmModal_KeyboardShortcuts(t *testing.T) {
    // Test 'y' for yes, 'n' for no shortcuts
}
```

### 3.2 Error Modal Tests

```go
func TestErrorModal_DisplayAndDismiss(t *testing.T) {
    // Test error modal appears on error
    // Test dismissing with Enter or Escape
}
```

### 3.3 Stack Selector Tests

```go
func TestStackSelector_Navigation(t *testing.T) {
    // Test selecting different stacks
    // Test creating new stack option
}
```

### 3.4 Help Modal Tests

```go
func TestHelpModal_ToggleWithQuestionMark(t *testing.T) {
    m := createTestModel(t, withResources(testResources))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    waitForContent(t, tm, "resource", 3*time.Second)
    
    // Open help
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
    waitForContent(t, tm, "Keybindings", 2*time.Second)
    
    // Close help
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}
```

---

## Part 4: Golden File Tests (View Regression)

### 4.1 Setup Golden File Testing

```go
// TestView_ResourceList_Golden tests resource list rendering
func TestView_ResourceList_Golden(t *testing.T) {
    resources := []ResourceState{
        {URN: "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev", Type: "pulumi:pulumi:Stack"},
        {URN: "urn:pulumi:dev::test::aws:s3:Bucket::mybucket", Type: "aws:s3:Bucket"},
        {URN: "urn:pulumi:dev::test::aws:lambda:Function::myfunc", Type: "aws:lambda:Function"},
    }
    
    m := createTestModel(t, withResources(resources))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
    
    // Wait for stable state
    waitForContent(t, tm, "mybucket", 3*time.Second)
    
    // Give time for final render
    time.Sleep(100 * time.Millisecond)
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    
    out, err := io.ReadAll(tm.FinalOutput(t))
    require.NoError(t, err)
    
    teatest.RequireEqualOutput(t, out)
}
```

### 4.2 Recommended Golden File Tests

| Test Name | View State | Purpose |
|-----------|------------|---------|
| `TestView_ResourceList_Golden` | Resource list with multiple items | Main view regression |
| `TestView_ResourceTree_Golden` | Tree view with hierarchy | Tree rendering |
| `TestView_PreviewDiff_Golden` | Preview with changes | Diff display |
| `TestView_OperationProgress_Golden` | Operation in progress | Progress indicators |
| `TestView_ErrorState_Golden` | Error displayed | Error styling |
| `TestView_EmptyState_Golden` | No resources | Empty state message |

### 4.3 Updating Golden Files

When intentionally changing views, update golden files:

```bash
# Update all golden files
go test -tags=integration ./cmd/p5 -update

# Or set environment variable
TEATEST_UPDATE=true go test -tags=integration ./cmd/p5
```

---

## Part 5: Edge Case and Error Handling Tests

### 5.1 Cancellation Tests

```go
func TestOperation_CancelDuringPreview(t *testing.T) {
    // Send slow preview events
    // Press Ctrl+C during preview
    // Verify graceful cancellation
}

func TestOperation_CancelDuringExecution(t *testing.T) {
    // Start operation
    // Press Ctrl+C during execution
    // Verify cancellation message and state
}
```

### 5.2 Resize Handling Tests

```go
func TestResize_DuringOperation(t *testing.T) {
    m := createTestModel(t, withStartView("up"))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    // Start preview
    waitForContent(t, tm, "preview", 3*time.Second)
    
    // Send resize message
    tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
    
    // Verify app handles resize gracefully
    time.Sleep(100 * time.Millisecond)
    
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t)
}
```

### 5.3 Rapid Input Tests

```go
func TestRapidKeyPresses(t *testing.T) {
    m := createTestModel(t, withResources(manyResources))
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
    
    waitForContent(t, tm, "resource", 3*time.Second)
    
    // Rapid navigation
    for i := 0; i < 20; i++ {
        tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
    }
    for i := 0; i < 10; i++ {
        tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
    }
    
    // Should not crash or hang
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}
```

---

## Part 6: Test Execution

### 6.1 Running Integration Tests

```bash
# Run integration tests only
go test -tags=integration ./cmd/p5 -v

# Run with timeout
go test -tags=integration ./cmd/p5 -v -timeout=5m

# Run specific test
go test -tags=integration ./cmd/p5 -v -run TestPreviewFlow

# Update golden files
go test -tags=integration ./cmd/p5 -update
```

### 6.2 CI Configuration

Add to GitHub Actions workflow:

```yaml
jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run integration tests
        run: go test -tags=integration ./cmd/p5 -v -timeout=5m
        env:
          # Force ASCII for consistent golden files
          TERM: dumb
```

---

## Part 7: Implementation Checklist

### Phase 1: Infrastructure (Priority: High)
- [ ] Add teatest dependency to go.mod
- [ ] Create `cmd/p5/integration_test.go` with build tag
- [ ] Add color profile initialization for tests
- [ ] Create test helper functions
- [ ] Configure `.gitattributes` for golden files

### Phase 2: Core Flow Tests (Priority: High)
- [ ] Initialization flow tests (success, workspace invalid, plugin failure)
- [ ] Navigation tests (j/k, Tab, view toggle)
- [ ] Preview flow tests (success, with errors, cancellation)
- [ ] Up operation flow tests (confirm, execute, cancel)

### Phase 3: Modal Tests (Priority: Medium)
- [ ] Confirm modal navigation
- [ ] Error modal display/dismiss
- [ ] Help modal toggle
- [ ] Import modal workflow
- [ ] Stack selector navigation

### Phase 4: Golden File Tests (Priority: Medium)
- [ ] Resource list view
- [ ] Resource tree view
- [ ] Preview diff view
- [ ] Operation progress view
- [ ] Error state view

### Phase 5: Edge Cases (Priority: Low)
- [ ] Cancellation during operations
- [ ] Window resize handling
- [ ] Rapid input handling
- [ ] Empty state handling

---

## Estimated Effort

| Phase | Tests | Estimated Time |
|-------|-------|----------------|
| Phase 1: Infrastructure | N/A | 2-3 hours |
| Phase 2: Core Flows | 8-12 | 4-6 hours |
| Phase 3: Modals | 6-8 | 3-4 hours |
| Phase 4: Golden Files | 5-6 | 2-3 hours |
| Phase 5: Edge Cases | 4-6 | 2-3 hours |
| **Total** | **23-32** | **13-19 hours** |

---

## References

- [teatest package documentation](https://pkg.go.dev/github.com/charmbracelet/x/exp/teatest)
- [Official teatest tutorial](https://charm.land/blog/teatest/)
- [teatest example repository](https://github.com/caarlos0/teatest-example)
- [VHS for visual testing](https://github.com/charmbracelet/vhs)
