# Pulumi Automation API

p5 uses the Pulumi Automation API for all stack operations except state deletion.

## Package

```go
import "github.com/pulumi/pulumi/sdk/v3/auto"
```

## Interfaces

p5 abstracts Pulumi operations behind interfaces for testability:

| Interface | Purpose | Location |
|-----------|---------|----------|
| `StackOperator` | Mutation operations (preview, up, refresh, destroy) | `internal/pulumi/interfaces.go` |
| `StackReader` | Read-only queries (resources, history, stacks) | `internal/pulumi/interfaces.go` |
| `WorkspaceReader` | Workspace-level queries (project info, workspaces) | `internal/pulumi/interfaces.go` |
| `StackInitializer` | Stack creation | `internal/pulumi/interfaces.go` |
| `ResourceImporter` | Import and state operations | `internal/pulumi/interfaces.go` |

## Operations

### Preview
```go
Preview(ctx, workDir, stackName, opType, opts) <-chan PreviewEvent
```
Runs `stack.Preview()`, `stack.PreviewRefresh()`, or `stack.PreviewDestroy()` based on operation type. Returns event channel for streaming results.

### Execute
```go
Up(ctx, workDir, stackName, opts) <-chan OperationEvent
Refresh(ctx, workDir, stackName, opts) <-chan OperationEvent
Destroy(ctx, workDir, stackName, opts) <-chan OperationEvent
```
Execute operations with event streaming.

### Read
```go
GetResources(ctx, workDir, stackName, opts) ([]ResourceInfo, error)
GetHistory(ctx, workDir, stackName, pageSize, page, opts) ([]UpdateSummary, error)
GetStacks(ctx, workDir, opts) ([]StackInfo, error)
```
Query stack state and history via `stack.Export()` and `stack.History()`.

### Import
```go
Import(ctx, workDir, stackName, resourceType, resourceName, importID, parentURN, opts) (*CommandResult, error)
```
Import existing cloud resources via `stack.ImportResources()`.

## Event Streaming

Operations use Pulumi's `events.EngineEvent` for real-time progress. Events are processed in `internal/pulumi/preview.go` and `internal/pulumi/operations.go`, converting to internal types:

- `PreviewEvent`: URN, operation type, resource metadata, inputs/outputs diff
- `OperationEvent`: Same as preview plus execution status (pending, running, success, failed)

## Workspace

`auto.NewLocalWorkspace()` creates workspaces for stack operations. Configuration:
- `WorkDir`: Pulumi project directory
- `EnvVars`: Environment variables from plugin authentication
- Project and stack settings from `Pulumi.yaml` and `Pulumi.{stack}.yaml`
