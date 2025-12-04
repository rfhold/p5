# Pulumi Automation API Go SDK

Reference documentation for building TUI applications with Pulumi Automation API.

## Overview

The Automation API allows programmatic control of Pulumi operations without shelling out to the CLI. Key benefits:
- Direct Go API for stack operations
- Event streaming for real-time progress updates
- Context-based cancellation
- Programmatic configuration management

## Stack Creation Patterns

Three ways to create/select a stack:

```go
// Option 1: Inline source (function-based program)
stack, err := auto.NewStackInlineSource(ctx, "org/proj/stack", "myProj", 
    func(pCtx *pulumi.Context) error {
        // define resources here
        return nil
    })

// Option 2: Local source (existing Pulumi.yaml on disk)
stack, err := auto.NewStackLocalSource(ctx, "org/proj/stack", "/path/to/project")

// Option 3: Remote source (git repo)
stack, err := auto.NewStackRemoteSource(ctx, "org/proj/stack", auto.GitRepo{
    URL: "https://github.com/org/repo.git",
})
```

### Stack Selection Variants

```go
// UpsertStack* - create if not exists, select if exists
stack, err := auto.UpsertStackLocalSource(ctx, "dev", "/path/to/project")

// SelectStack* - select existing only (errors if not found)
stack, err := auto.SelectStackLocalSource(ctx, "dev", "/path/to/project")
```

## Stack Operations

### Function Signatures

```go
func (s *Stack) Up(ctx context.Context, opts ...optup.Option) (UpResult, error)
func (s *Stack) Preview(ctx context.Context, opts ...optpreview.Option) (PreviewResult, error)
func (s *Stack) Refresh(ctx context.Context, opts ...optrefresh.Option) (RefreshResult, error)
func (s *Stack) Destroy(ctx context.Context, opts ...optdestroy.Option) (DestroyResult, error)
```

### Common Options

```go
// Up options
optup.EventStreams(ch)      // Stream events to channel
optup.ProgressStreams(w)    // Write progress to io.Writer
optup.Target([]string{})    // Target specific resources
optup.Parallel(n)           // Parallelism limit
optup.Message("msg")        // Update message

// Similar options exist for preview, refresh, destroy
// Import: github.com/pulumi/pulumi/sdk/v3/go/auto/optup (etc.)
```

## Event Streaming

Events are streamed via a channel passed as an option:

```go
import "github.com/pulumi/pulumi/sdk/v3/go/auto/events"

// Create event channel
eventChannel := make(chan events.EngineEvent)

// Start goroutine to consume events BEFORE calling operation
go func() {
    for event := range eventChannel {
        // Process event - check which field is non-nil
        if event.ResourcePreEvent != nil {
            // Resource operation starting
        }
        if event.ResOutputsEvent != nil {
            // Resource operation completed
        }
        if event.DiagnosticEvent != nil {
            // Log/diagnostic message
        }
        if event.SummaryEvent != nil {
            // Operation summary (at end)
        }
    }
}()

// Pass channel to operation
result, err := stack.Up(ctx, optup.EventStreams(eventChannel))
// Channel is closed when operation completes
```

### Event Types (EngineEvent fields)

The `events.EngineEvent` struct has optional fields - check which is non-nil:

| Field | Description |
|-------|-------------|
| `CancelEvent` | Operation was cancelled |
| `StdoutEvent` | Stdout from program |
| `DiagnosticEvent` | Log messages, warnings, errors |
| `PreludeEvent` | Operation starting, lists config |
| `SummaryEvent` | Final summary (resources created/updated/deleted) |
| `ResourcePreEvent` | Resource operation starting |
| `ResOutputsEvent` | Resource operation completed |
| `ResOpFailedEvent` | Resource operation failed |
| `PolicyEvent` | Policy violation |

### Event Stream Options by Operation

```go
optup.EventStreams(ch chan<- events.EngineEvent)
optpreview.EventStreams(ch chan<- events.EngineEvent)
optrefresh.EventStreams(ch chan<- events.EngineEvent)
optdestroy.EventStreams(ch chan<- events.EngineEvent)
```

## Cancellation

All operations accept `context.Context` for cancellation:

```go
// Create cancellable context
ctx, cancel := context.WithCancel(context.Background())

// Run operation in goroutine
go func() {
    result, err := stack.Up(ctx, optup.EventStreams(events))
    if err != nil {
        // Check if cancelled
        if ctx.Err() == context.Canceled {
            // Operation was cancelled
        }
    }
}()

// Cancel when needed (e.g., user presses Ctrl+C)
cancel()
```

### Forceful Cancellation

```go
// WARNING: May leave stack in inconsistent state
err := stack.Cancel(ctx)
```

## Reading Stack State

### Stack Outputs

```go
// After Up completes
result, err := stack.Up(ctx)
if err != nil { return err }

// Access outputs from result
for key, output := range result.Outputs {
    fmt.Printf("%s = %v (secret: %v)\n", key, output.Value, output.Secret)
}

// Or get outputs directly
outputs, err := stack.Outputs(ctx)
```

### Stack Info

```go
// Get stack summary
info, err := stack.Info(ctx)
// info.Current, info.ResourceCount, info.URL, etc.

// Export full state
state, err := stack.Export(ctx)
```

## Workspace Operations

The workspace manages project settings, configuration, and plugins:

```go
// Create workspace
ws, err := auto.NewLocalWorkspace(ctx, auto.WorkDir("/path/to/project"))

// Project settings
project, err := ws.ProjectSettings(ctx)
fmt.Println(project.Name, project.Runtime.Name())

// List stacks
stacks, err := ws.ListStacks(ctx)
for _, s := range stacks {
    fmt.Printf("%s (current: %v)\n", s.Name, s.Current)
}

// Stack configuration
cfg, err := ws.GetAllConfig(ctx, "dev")
ws.SetConfig(ctx, "dev", "key", auto.ConfigValue{Value: "value"})
ws.SetConfig(ctx, "dev", "secret", auto.ConfigValue{Value: "val", Secret: true})
```

## Complete Example: TUI Integration

```go
func runPulumiUp(ctx context.Context, updateCh chan<- UIUpdate) error {
    // 1. Create/select stack
    stack, err := auto.UpsertStackLocalSource(ctx, "dev", ".")
    if err != nil {
        return fmt.Errorf("failed to create stack: %w", err)
    }

    // 2. Setup event streaming
    events := make(chan events.EngineEvent)
    go func() {
        for e := range events {
            // Convert to UI updates
            if e.ResourcePreEvent != nil {
                updateCh <- UIUpdate{
                    Type:     "resource_start",
                    Resource: e.ResourcePreEvent.Metadata.URN,
                    Op:       string(e.ResourcePreEvent.Metadata.Op),
                }
            }
            if e.ResOutputsEvent != nil {
                updateCh <- UIUpdate{
                    Type:     "resource_done",
                    Resource: e.ResOutputsEvent.Metadata.URN,
                }
            }
            if e.DiagnosticEvent != nil {
                updateCh <- UIUpdate{
                    Type:    "diagnostic",
                    Message: e.DiagnosticEvent.Message,
                    Level:   string(e.DiagnosticEvent.Severity),
                }
            }
        }
    }()

    // 3. Run operation
    result, err := stack.Up(ctx, optup.EventStreams(events))
    if err != nil {
        if auto.IsConcurrentUpdateError(err) {
            return fmt.Errorf("stack is locked by another operation")
        }
        return err
    }

    // 4. Send completion
    updateCh <- UIUpdate{
        Type:    "complete",
        Summary: fmt.Sprintf("Created: %d, Updated: %d, Deleted: %d",
            result.Summary.ResourceChanges["create"],
            result.Summary.ResourceChanges["update"],
            result.Summary.ResourceChanges["delete"]),
    }

    return nil
}
```

## Error Handling

```go
// Check specific error types
if auto.IsConcurrentUpdateError(err) {
    // Stack is locked by another operation
}

if auto.IsSelectStack404Error(err) {
    // Stack doesn't exist
}

if auto.IsCreateStack409Error(err) {
    // Stack already exists
}
```

## References

- [pkg.go.dev/github.com/pulumi/pulumi/sdk/v3/go/auto](https://pkg.go.dev/github.com/pulumi/pulumi/sdk/v3/go/auto)
- [Pulumi Automation API Docs](https://www.pulumi.com/docs/using-pulumi/automation-api/)
