# Pulumi Automation API Integration Testing Plan

This document outlines the strategy for adding integration tests that exercise real Pulumi operations in the p5 TUI application.

## Overview

Our current unit tests use fake implementations (`FakeStackOperator`, `FakeStackReader`, etc.) to test TUI logic in isolation. Integration tests will validate that our Pulumi wrapper code correctly interacts with the real Pulumi CLI and Automation API.

### Testing Pyramid for Pulumi Operations

| Layer | Approach | Speed | What It Tests |
|-------|----------|-------|---------------|
| Unit | Fake implementations | Fast (ms) | TUI logic, event handling, state transitions |
| Integration | Local backend + real CLI | Medium (1-10s) | Pulumi wrapper code, event streams, config handling |
| E2E | Real cloud providers | Slow (minutes) | Actual infrastructure operations (CI only) |

---

## Prerequisites

### 1. Pulumi CLI Installation

Integration tests require the Pulumi CLI to be installed:

```bash
# macOS
brew install pulumi

# Or via script
curl -fsSL https://get.pulumi.com | sh
```

### 2. Test Dependencies

```bash
# Pulumi testing utilities
go get github.com/pulumi/pulumi/sdk/v3/go/common/testing
```

### 3. Environment Configuration

Integration tests use local state and passphrase encryption to avoid external dependencies:

```bash
# No cloud credentials needed
export PULUMI_CONFIG_PASSPHRASE="test-passphrase-12345"
```

---

## Part 1: Test Infrastructure Setup

### 1.1 Create Integration Test File

Create `internal/pulumi/integration_test.go` with build tag:

```go
//go:build integration

package pulumi

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/pulumi/pulumi/sdk/v3/go/auto"
    ptesting "github.com/pulumi/pulumi/sdk/v3/go/common/testing"
    "github.com/stretchr/testify/require"
)

// Integration tests for Pulumi Automation API wrapper code
```

### 1.2 Test Stack Helper

```go
// TestStack wraps a Pulumi stack with cleanup functionality
type TestStack struct {
    Stack   auto.Stack
    TmpDir  string
    t       *testing.T
    cleaned bool
}

// SetupTestStack creates an isolated test stack with local file backend
func SetupTestStack(t *testing.T, projectName string, program auto.RunFunc) *TestStack {
    t.Helper()
    
    ctx := context.Background()
    
    // Create temporary directory for state
    tmpDir, err := os.MkdirTemp("", "pulumi-test-*")
    require.NoError(t, err, "failed to create temp dir")
    
    // Generate unique stack name
    stackName := ptesting.RandomStackName()
    
    // Create stack with local backend
    s, err := auto.NewStackInlineSource(ctx, stackName, projectName, program,
        auto.EnvVars(map[string]string{
            "PULUMI_BACKEND_URL":       "file://" + tmpDir,
            "PULUMI_CONFIG_PASSPHRASE": "test-passphrase-12345",
        }),
        auto.SecretsProvider("passphrase"),
    )
    if err != nil {
        os.RemoveAll(tmpDir)
        t.Fatalf("failed to create test stack: %v", err)
    }
    
    ts := &TestStack{
        Stack:  s,
        TmpDir: tmpDir,
        t:      t,
    }
    
    // Register cleanup
    t.Cleanup(ts.Cleanup)
    
    return ts
}

// Cleanup destroys the stack and removes temporary files
func (ts *TestStack) Cleanup() {
    if ts.cleaned {
        return
    }
    ts.cleaned = true
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    // Destroy resources (ignore errors - may not have any)
    ts.Stack.Destroy(ctx)
    
    // Remove stack from backend
    ts.Stack.Workspace().RemoveStack(ctx, ts.Stack.Name())
    
    // Remove temp directory
    os.RemoveAll(ts.TmpDir)
}

// Name returns the stack's full name
func (ts *TestStack) Name() string {
    return ts.Stack.Name()
}
```

### 1.3 Test Program Helpers

```go
// Simple test program that creates a random ID resource
func simpleProgram(ctx *pulumi.Context) error {
    _, err := random.NewRandomId(ctx, "test-id", &random.RandomIdArgs{
        ByteLength: pulumi.Int(8),
    })
    return err
}

// Program that creates multiple resources with dependencies
func multiResourceProgram(ctx *pulumi.Context) error {
    id, err := random.NewRandomId(ctx, "base-id", &random.RandomIdArgs{
        ByteLength: pulumi.Int(8),
    })
    if err != nil {
        return err
    }
    
    _, err = random.NewRandomString(ctx, "derived-string", &random.RandomStringArgs{
        Length: pulumi.Int(16),
        Keepers: pulumi.StringMap{
            "base": id.Hex,
        },
    })
    return err
}

// Program that will fail during preview
func failingProgram(ctx *pulumi.Context) error {
    return errors.New("intentional failure for testing")
}
```

### 1.4 Event Collection Helper

```go
// CollectEvents collects all events from a channel into a slice
func CollectEvents[T any](ch <-chan T) []T {
    var events []T
    for event := range ch {
        events = append(events, event)
    }
    return events
}

// CollectEventsAsync collects events in a goroutine and returns when channel closes
func CollectEventsAsync[T any](ch <-chan T) ([]T, *sync.WaitGroup) {
    var events []T
    var mu sync.Mutex
    var wg sync.WaitGroup
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        for event := range ch {
            mu.Lock()
            events = append(events, event)
            mu.Unlock()
        }
    }()
    
    return events, &wg
}
```

---

## Part 2: DefaultOperator Tests

### 2.1 Preview Operation Tests

```go
func TestDefaultOperator_Preview_Success(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "preview-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, err := operator.Preview(ctx)
    require.NoError(t, err)
    
    events := CollectEvents(eventCh)
    
    // Verify we got events
    require.NotEmpty(t, events, "expected preview events")
    
    // Verify we got a completion event
    var foundDone bool
    for _, e := range events {
        if e.Done {
            foundDone = true
            require.NoError(t, e.Error, "preview should succeed")
        }
    }
    require.True(t, foundDone, "expected Done event")
}

func TestDefaultOperator_Preview_WithChanges(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "preview-changes-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, err := operator.Preview(ctx)
    require.NoError(t, err)
    
    events := CollectEvents(eventCh)
    
    // Find step events
    var createSteps int
    for _, e := range events {
        if e.Step != nil && e.Step.Op == OpCreate {
            createSteps++
        }
    }
    
    // Should have at least one create (the random ID)
    require.GreaterOrEqual(t, createSteps, 1, "expected create steps")
}

func TestDefaultOperator_Preview_Cancellation(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "preview-cancel-test", simpleProgram)
    
    // Create cancellable context
    ctx, cancel := context.WithCancel(context.Background())
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, err := operator.Preview(ctx)
    require.NoError(t, err)
    
    // Cancel immediately
    cancel()
    
    // Drain events - should complete quickly
    done := make(chan struct{})
    go func() {
        CollectEvents(eventCh)
        close(done)
    }()
    
    select {
    case <-done:
        // Success - channel closed
    case <-time.After(30 * time.Second):
        t.Fatal("preview did not cancel in time")
    }
}

func TestDefaultOperator_Preview_ProgramError(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "preview-error-test", failingProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, err := operator.Preview(ctx)
    require.NoError(t, err) // Starting preview succeeds
    
    events := CollectEvents(eventCh)
    
    // Should have error in events
    var foundError bool
    for _, e := range events {
        if e.Error != nil {
            foundError = true
        }
    }
    require.True(t, foundError, "expected error event for failing program")
}
```

### 2.2 Up Operation Tests

```go
func TestDefaultOperator_Up_Success(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "up-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, err := operator.Up(ctx)
    require.NoError(t, err)
    
    events := CollectEvents(eventCh)
    
    // Verify completion
    var foundDone bool
    var success bool
    for _, e := range events {
        if e.Done {
            foundDone = true
            success = e.Success
        }
    }
    
    require.True(t, foundDone, "expected Done event")
    require.True(t, success, "up operation should succeed")
}

func TestDefaultOperator_Up_ThenDestroy(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "up-destroy-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    // First, run up
    upCh, err := operator.Up(ctx)
    require.NoError(t, err)
    CollectEvents(upCh)
    
    // Now destroy
    destroyCh, err := operator.Destroy(ctx)
    require.NoError(t, err)
    
    events := CollectEvents(destroyCh)
    
    // Verify destruction completed
    var foundDelete bool
    for _, e := range events {
        if e.Step != nil && e.Step.Op == OpDelete {
            foundDelete = true
        }
    }
    require.True(t, foundDelete, "expected delete operations")
}

func TestDefaultOperator_Up_Idempotent(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "up-idempotent-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    // First up
    upCh1, _ := operator.Up(ctx)
    CollectEvents(upCh1)
    
    // Second up - should have no changes
    upCh2, _ := operator.Up(ctx)
    events := CollectEvents(upCh2)
    
    // Count actual changes (not same operations)
    var changes int
    for _, e := range events {
        if e.Step != nil && e.Step.Op != OpSame {
            changes++
        }
    }
    require.Zero(t, changes, "second up should have no changes")
}
```

### 2.3 Refresh Operation Tests

```go
func TestDefaultOperator_Refresh_Success(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "refresh-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    // First deploy
    upCh, _ := operator.Up(ctx)
    CollectEvents(upCh)
    
    // Then refresh
    refreshCh, err := operator.Refresh(ctx)
    require.NoError(t, err)
    
    events := CollectEvents(refreshCh)
    
    // Should complete successfully
    var foundDone bool
    for _, e := range events {
        if e.Done {
            foundDone = true
            require.True(t, e.Success)
        }
    }
    require.True(t, foundDone)
}
```

---

## Part 3: DefaultReader Tests

### 3.1 Stack State Reading Tests

```go
func TestDefaultReader_GetResources_Empty(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "read-empty-test", simpleProgram)
    ctx := context.Background()
    
    reader := NewDefaultReader(ts.Stack)
    
    resources, err := reader.GetResources(ctx)
    require.NoError(t, err)
    
    // Empty stack should have only the stack resource
    require.Len(t, resources, 1)
    require.Contains(t, resources[0].Type, "pulumi:pulumi:Stack")
}

func TestDefaultReader_GetResources_AfterUp(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "read-after-up-test", multiResourceProgram)
    ctx := context.Background()
    
    // Deploy first
    operator := NewDefaultOperator(ts.Stack)
    upCh, _ := operator.Up(ctx)
    CollectEvents(upCh)
    
    // Now read
    reader := NewDefaultReader(ts.Stack)
    resources, err := reader.GetResources(ctx)
    require.NoError(t, err)
    
    // Should have stack + 2 resources
    require.GreaterOrEqual(t, len(resources), 3)
    
    // Verify resource types
    var foundRandomId, foundRandomString bool
    for _, r := range resources {
        if strings.Contains(r.Type, "RandomId") {
            foundRandomId = true
        }
        if strings.Contains(r.Type, "RandomString") {
            foundRandomString = true
        }
    }
    require.True(t, foundRandomId)
    require.True(t, foundRandomString)
}

func TestDefaultReader_GetOutputs(t *testing.T) {
    t.Parallel()
    
    // Program that exports outputs
    program := func(ctx *pulumi.Context) error {
        id, _ := random.NewRandomId(ctx, "test", &random.RandomIdArgs{
            ByteLength: pulumi.Int(8),
        })
        ctx.Export("randomHex", id.Hex)
        return nil
    }
    
    ts := SetupTestStack(t, "outputs-test", program)
    ctx := context.Background()
    
    // Deploy
    operator := NewDefaultOperator(ts.Stack)
    upCh, _ := operator.Up(ctx)
    CollectEvents(upCh)
    
    // Read outputs
    reader := NewDefaultReader(ts.Stack)
    outputs, err := reader.GetOutputs(ctx)
    require.NoError(t, err)
    
    require.Contains(t, outputs, "randomHex")
    require.NotEmpty(t, outputs["randomHex"])
}
```

---

## Part 4: DefaultWorkspace Tests

### 4.1 Workspace Validation Tests

```go
func TestDefaultWorkspace_IsValidWorkspace(t *testing.T) {
    // Test with actual test/simple directory
    ws := NewDefaultWorkspaceReader()
    
    valid, err := ws.IsValidWorkspace("../../test/simple")
    require.NoError(t, err)
    require.True(t, valid)
}

func TestDefaultWorkspace_IsValidWorkspace_Invalid(t *testing.T) {
    ws := NewDefaultWorkspaceReader()
    
    // Test with directory that has no Pulumi.yaml
    tmpDir, _ := os.MkdirTemp("", "invalid-*")
    defer os.RemoveAll(tmpDir)
    
    valid, err := ws.IsValidWorkspace(tmpDir)
    require.NoError(t, err)
    require.False(t, valid)
}

func TestDefaultWorkspace_ListStacks(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "list-stacks-test", simpleProgram)
    ctx := context.Background()
    
    ws := ts.Stack.Workspace()
    
    stacks, err := ws.ListStacks(ctx)
    require.NoError(t, err)
    
    // Should contain our test stack
    var found bool
    for _, s := range stacks {
        if s.Name == ts.Name() {
            found = true
        }
    }
    require.True(t, found, "test stack should be in list")
}
```

### 4.2 Configuration Tests

```go
func TestDefaultWorkspace_SetAndGetConfig(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "config-test", simpleProgram)
    ctx := context.Background()
    
    // Set config
    err := ts.Stack.SetConfig(ctx, "testKey", auto.ConfigValue{
        Value: "testValue",
    })
    require.NoError(t, err)
    
    // Get config
    cfg, err := ts.Stack.GetConfig(ctx, "testKey")
    require.NoError(t, err)
    require.Equal(t, "testValue", cfg.Value)
}

func TestDefaultWorkspace_SetSecretConfig(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "secret-config-test", simpleProgram)
    ctx := context.Background()
    
    // Set secret config
    err := ts.Stack.SetConfig(ctx, "secretKey", auto.ConfigValue{
        Value:  "secretValue",
        Secret: true,
    })
    require.NoError(t, err)
    
    // Get config - should be marked as secret
    cfg, err := ts.Stack.GetConfig(ctx, "secretKey")
    require.NoError(t, err)
    require.True(t, cfg.Secret)
}
```

---

## Part 5: Event Stream Tests

### 5.1 Event Type Coverage

```go
func TestEventStream_AllEventTypes(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "events-test", multiResourceProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, _ := operator.Up(ctx)
    events := CollectEvents(eventCh)
    
    // Categorize events
    var (
        stepEvents     int
        progressEvents int
        diagnostics    int
        doneEvent      bool
    )
    
    for _, e := range events {
        if e.Step != nil {
            stepEvents++
        }
        if e.Progress != "" {
            progressEvents++
        }
        if e.Diagnostic != "" {
            diagnostics++
        }
        if e.Done {
            doneEvent = true
        }
    }
    
    t.Logf("Events: steps=%d, progress=%d, diagnostics=%d, done=%v",
        stepEvents, progressEvents, diagnostics, doneEvent)
    
    require.True(t, doneEvent, "must have done event")
    require.Greater(t, stepEvents, 0, "should have step events")
}

func TestEventStream_StepOperationTypes(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "step-ops-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    // Up - should have create
    upCh, _ := operator.Up(ctx)
    upEvents := CollectEvents(upCh)
    
    var hasCreate bool
    for _, e := range upEvents {
        if e.Step != nil && e.Step.Op == OpCreate {
            hasCreate = true
        }
    }
    require.True(t, hasCreate, "up should have create operations")
    
    // Second up - should have same
    up2Ch, _ := operator.Up(ctx)
    up2Events := CollectEvents(up2Ch)
    
    var hasSame bool
    for _, e := range up2Events {
        if e.Step != nil && e.Step.Op == OpSame {
            hasSame = true
        }
    }
    require.True(t, hasSame, "second up should have same operations")
    
    // Destroy - should have delete
    destroyCh, _ := operator.Destroy(ctx)
    destroyEvents := CollectEvents(destroyCh)
    
    var hasDelete bool
    for _, e := range destroyEvents {
        if e.Step != nil && e.Step.Op == OpDelete {
            hasDelete = true
        }
    }
    require.True(t, hasDelete, "destroy should have delete operations")
}
```

### 5.2 Event Ordering Tests

```go
func TestEventStream_Ordering(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "event-order-test", multiResourceProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    eventCh, _ := operator.Up(ctx)
    events := CollectEvents(eventCh)
    
    // Done should be last
    require.True(t, events[len(events)-1].Done, "Done should be last event")
    
    // Verify parent resources are created before children
    // (specific to multiResourceProgram structure)
    var baseIdIndex, derivedIndex int = -1, -1
    for i, e := range events {
        if e.Step == nil {
            continue
        }
        if strings.Contains(e.Step.URN, "base-id") && e.Step.Op == OpCreate {
            baseIdIndex = i
        }
        if strings.Contains(e.Step.URN, "derived-string") && e.Step.Op == OpCreate {
            derivedIndex = i
        }
    }
    
    if baseIdIndex >= 0 && derivedIndex >= 0 {
        require.Less(t, baseIdIndex, derivedIndex,
            "base resource should be created before derived")
    }
}
```

---

## Part 6: Import Operation Tests

### 6.1 Import Workflow Tests

```go
func TestDefaultImporter_Import_Success(t *testing.T) {
    // Note: Import tests require resources that actually exist
    // This is typically tested with provider-specific resources
    // For unit testing, we rely on the FakeImporter
    t.Skip("Import tests require actual cloud resources")
}

func TestDefaultImporter_GenerateCode(t *testing.T) {
    // Test code generation for imported resources
    t.Skip("Requires imported resources")
}
```

---

## Part 7: History Tests

### 7.1 Stack History Tests

```go
func TestHistory_AfterOperations(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "history-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    // Run up
    upCh, _ := operator.Up(ctx)
    CollectEvents(upCh)
    
    // Check history
    history, err := ts.Stack.History(ctx, 10, 1)
    require.NoError(t, err)
    require.NotEmpty(t, history)
    
    // Most recent should be update
    require.Equal(t, "update", history[0].Kind)
}

func TestHistory_MultipleOperations(t *testing.T) {
    t.Parallel()
    
    ts := SetupTestStack(t, "multi-history-test", simpleProgram)
    ctx := context.Background()
    
    operator := NewDefaultOperator(ts.Stack)
    
    // Up, refresh, destroy
    upCh, _ := operator.Up(ctx)
    CollectEvents(upCh)
    
    refreshCh, _ := operator.Refresh(ctx)
    CollectEvents(refreshCh)
    
    destroyCh, _ := operator.Destroy(ctx)
    CollectEvents(destroyCh)
    
    // Check history - should have 3 entries
    history, err := ts.Stack.History(ctx, 10, 1)
    require.NoError(t, err)
    require.GreaterOrEqual(t, len(history), 3)
}
```

---

## Part 8: Parallel Execution Tests

### 8.1 Concurrent Stack Operations

```go
func TestParallel_MultipleStacks(t *testing.T) {
    t.Parallel()
    
    const numStacks = 4
    var wg sync.WaitGroup
    errors := make(chan error, numStacks)
    
    for i := 0; i < numStacks; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            
            ts := SetupTestStack(t, fmt.Sprintf("parallel-test-%d", idx), simpleProgram)
            ctx := context.Background()
            
            operator := NewDefaultOperator(ts.Stack)
            
            upCh, err := operator.Up(ctx)
            if err != nil {
                errors <- err
                return
            }
            
            events := CollectEvents(upCh)
            for _, e := range events {
                if e.Error != nil {
                    errors <- e.Error
                    return
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    for err := range errors {
        t.Errorf("parallel operation failed: %v", err)
    }
}
```

---

## Part 9: Test Execution

### 9.1 Running Integration Tests

```bash
# Run all integration tests
go test -tags=integration ./internal/pulumi -v

# Run with longer timeout (operations can be slow)
go test -tags=integration ./internal/pulumi -v -timeout=10m

# Run specific test
go test -tags=integration ./internal/pulumi -v -run TestDefaultOperator_Preview

# Run with race detection
go test -tags=integration ./internal/pulumi -v -race

# Skip cleanup for debugging (manually clean up later)
SKIP_CLEANUP=1 go test -tags=integration ./internal/pulumi -v
```

### 9.2 CI Configuration

```yaml
# .github/workflows/integration.yml
name: Integration Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  pulumi-integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - uses: pulumi/setup-pulumi@v2
      
      - name: Run Pulumi integration tests
        run: |
          go test -tags=integration ./internal/pulumi -v -timeout=10m
        env:
          PULUMI_CONFIG_PASSPHRASE: ${{ secrets.PULUMI_TEST_PASSPHRASE }}
```

### 9.3 Local Development Setup

Create `scripts/integration-test.sh`:

```bash
#!/bin/bash
set -e

export PULUMI_CONFIG_PASSPHRASE="test-passphrase-12345"

echo "Running Pulumi integration tests..."
go test -tags=integration ./internal/pulumi -v -timeout=10m "$@"

echo "Running Bubble Tea integration tests..."
go test -tags=integration ./cmd/p5 -v -timeout=5m "$@"
```

---

## Part 10: Mock PulumiCommand Pattern

For faster tests that don't need the full CLI, use the mock command pattern from Pulumi's own SDK:

### 10.1 Mock Command Implementation

```go
// mockPulumiCommand implements auto.PulumiCommand for testing
type mockPulumiCommand struct {
    version      semver.Version
    responses    map[string]mockResponse
    capturedArgs [][]string
    mu           sync.Mutex
}

type mockResponse struct {
    stdout   string
    stderr   string
    exitCode int
    err      error
}

func newMockPulumiCommand() *mockPulumiCommand {
    return &mockPulumiCommand{
        version:   semver.MustParse("3.100.0"),
        responses: make(map[string]mockResponse),
    }
}

func (m *mockPulumiCommand) Version() semver.Version {
    return m.version
}

func (m *mockPulumiCommand) Run(ctx context.Context,
    workdir string,
    stdin io.Reader,
    additionalOutput []io.Writer,
    additionalErrorOutput []io.Writer,
    additionalEnv []string,
    args ...string,
) (string, string, int, error) {
    m.mu.Lock()
    m.capturedArgs = append(m.capturedArgs, args)
    m.mu.Unlock()
    
    // Build key from args
    key := strings.Join(args, " ")
    
    if resp, ok := m.responses[key]; ok {
        return resp.stdout, resp.stderr, resp.exitCode, resp.err
    }
    
    // Default response
    return "", "", 0, nil
}

func (m *mockPulumiCommand) OnCommand(args string, resp mockResponse) {
    m.responses[args] = resp
}
```

### 10.2 Usage Example

```go
func TestWorkspace_ListStacks_WithMock(t *testing.T) {
    mock := newMockPulumiCommand()
    mock.OnCommand("stack ls --json", mockResponse{
        stdout: `[{"name": "org/project/dev", "current": true}]`,
    })
    
    ws, err := auto.NewLocalWorkspace(ctx,
        auto.WorkDir("./testdata"),
        auto.Pulumi(mock),
    )
    require.NoError(t, err)
    
    stacks, err := ws.ListStacks(ctx)
    require.NoError(t, err)
    require.Len(t, stacks, 1)
    require.Equal(t, "org/project/dev", stacks[0].Name)
}
```

---

## Part 11: Implementation Checklist

### Phase 1: Infrastructure (Priority: High)
- [ ] Create `internal/pulumi/integration_test.go` with build tag
- [ ] Implement `SetupTestStack` helper with cleanup
- [ ] Create test program helpers (simple, multi-resource, failing)
- [ ] Add event collection helpers
- [ ] Create `scripts/integration-test.sh`

### Phase 2: DefaultOperator Tests (Priority: High)
- [ ] Preview success tests
- [ ] Preview with changes tests
- [ ] Preview cancellation tests
- [ ] Preview error handling tests
- [ ] Up success tests
- [ ] Up then destroy tests
- [ ] Up idempotency tests
- [ ] Refresh tests
- [ ] Destroy tests

### Phase 3: DefaultReader Tests (Priority: Medium)
- [ ] GetResources empty stack
- [ ] GetResources after up
- [ ] GetOutputs tests
- [ ] Resource property tests

### Phase 4: DefaultWorkspace Tests (Priority: Medium)
- [ ] Workspace validation tests
- [ ] Stack listing tests
- [ ] Config get/set tests
- [ ] Secret config tests

### Phase 5: Event Stream Tests (Priority: Medium)
- [ ] Event type coverage
- [ ] Step operation types
- [ ] Event ordering
- [ ] Error event handling

### Phase 6: Advanced Tests (Priority: Low)
- [ ] Parallel stack operations
- [ ] History tests
- [ ] Mock command pattern tests
- [ ] Import workflow tests (if applicable)

### Phase 7: CI Integration (Priority: Medium)
- [ ] GitHub Actions workflow
- [ ] Test secrets configuration
- [ ] Timeout configuration

---

## Estimated Effort

| Phase | Tests | Estimated Time |
|-------|-------|----------------|
| Phase 1: Infrastructure | N/A | 3-4 hours |
| Phase 2: DefaultOperator | 10-12 | 4-6 hours |
| Phase 3: DefaultReader | 4-6 | 2-3 hours |
| Phase 4: DefaultWorkspace | 4-6 | 2-3 hours |
| Phase 5: Event Streams | 4-6 | 2-3 hours |
| Phase 6: Advanced | 4-6 | 3-4 hours |
| Phase 7: CI | N/A | 1-2 hours |
| **Total** | **26-36** | **17-25 hours** |

---

## Test Data Requirements

### Required Test Programs

| Program | Resources | Purpose |
|---------|-----------|---------|
| `simpleProgram` | 1 random ID | Basic operations |
| `multiResourceProgram` | 2+ with deps | Dependency ordering |
| `failingProgram` | None (errors) | Error handling |
| `outputProgram` | 1 + exports | Output testing |

### No Cloud Credentials Required

All integration tests use:
- Local file backend (`file://tmpdir`)
- Passphrase secrets provider
- `random` provider (no cloud auth)

---

## References

- [Pulumi Automation API Go SDK](https://pkg.go.dev/github.com/pulumi/pulumi/sdk/v3/go/auto)
- [Pulumi SDK local_workspace_test.go](https://github.com/pulumi/pulumi/blob/master/sdk/go/auto/local_workspace_test.go)
- [Pulumi Testing Documentation](https://www.pulumi.com/docs/iac/guides/testing/)
- [automation-api-examples repository](https://github.com/pulumi/automation-api-examples)
