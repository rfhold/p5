# Preview

![Workflow Demo](../assets/workflow.gif)

Preview operations show what changes Pulumi would make without applying them.

## Operations

| Key | Operation | Description |
|-----|-----------|-------------|
| `u` | Up | Preview creating/updating resources |
| `d` | Destroy | Preview destroying resources |
| `r` | Refresh | Preview refreshing state from cloud |

## Flow

1. Press preview key (`u`/`d`/`r`)
2. Operation state transitions to `Starting`
3. View switches to `ViewPreview`
4. Collects target/replace/exclude flags from resource list
5. Calls `StackOperator.Preview()` returning event channel
6. Events stream in, updating resource list in real-time
7. Preview completes when `event.Done = true` or error

## Event Processing

Preview events contain:
- URN and operation type (create/update/delete/same/replace)
- Resource type and name
- Parent URN
- Inputs/outputs diff
- Sequence number

Events are converted to `ui.ResourceItem` and displayed in tree view.

## Header Summary

During preview, header shows running summary:
- Create count (green `+`)
- Update count (yellow `~`)
- Delete count (red `-`)
- Same count

## Cancellation

Press `Esc` during preview to cancel. Operation state transitions to `Cancelling` and context is cancelled.

## Resource Operations

| Operation | Symbol | Color |
|-----------|--------|-------|
| Create | `+` | Green |
| Update | `~` | Yellow |
| Delete | `-` | Red |
| Same | ` ` | Dim |
| Replace | `±` | Magenta |

## Related

- [Execute](execute.md) - Apply preview changes
- [Resource Targeting](resource-targetting.md) - Target specific resources
- [Details](details.md) - View resource details during preview
