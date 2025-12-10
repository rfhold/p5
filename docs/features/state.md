# State Management

View and manipulate Pulumi stack state.

![State Management](../assets/state.gif)

## Stack View

Default view showing current resources in the stack. Resources are displayed in a tree structure based on parent/child relationships.

## State Operations

### View Resources
Default view shows all resources from `stack.Export()`.

### Delete from State
Remove a resource from state without destroying the cloud resource.

| Key | Action |
|-----|--------|
| `x` | Delete selected resource from state |

Shows confirmation modal before deletion. Uses `pulumi state delete <urn>` CLI command.

## State Machine

Application tracks initialization state:

```
CheckingWorkspace → LoadingPlugins → LoadingStacks → SelectingStack → LoadingResources → Complete
```

And operation state:

```
Idle → Starting → Running → Cancelling → Complete/Error
```

## Resource Display

Resources show:
- Name (from URN)
- Type (provider:module/resource)
- Operation status (in preview/execute views)

Tree structure derived from parent URN relationships.

## Refresh

Press `r` to preview refresh operation, which reconciles state with actual cloud resources.

## Implementation

- `cmd/p5/state.go` - State types and transitions
- `internal/pulumi/resources.go` - Resource fetching
- `internal/ui/resourcelist.go` - Resource list display
- `internal/ui/resourcetree.go` - Tree rendering
