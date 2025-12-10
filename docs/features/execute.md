# Execute

![Workflow Demo](../assets/workflow.gif)

Apply changes to infrastructure.

## Operations

| Key | Operation | Description |
|-----|-----------|-------------|
| `ctrl+u` | Up | Execute update |
| `ctrl+r` | Refresh | Execute refresh |
| `ctrl+d` | Destroy | Execute destroy |

Note: `ctrl+` keys for execute, lowercase for preview.

## Confirmation

If not already on matching preview screen, shows confirmation modal:
- Displays operation type and target stack
- Requires explicit confirmation

If already viewing preview of same operation type, executes directly.

## Flow

1. Press execute key (`ctrl+u`/`ctrl+r`/`ctrl+d`)
2. Confirm if needed
3. Operation state transitions to `Starting`
4. View switches to `ViewExecute`
5. Calls appropriate `StackOperator` method
6. Events stream in, showing real-time progress
7. Resources show status: Pending → Running → Success/Failed

## Event Processing

Execute events contain same info as preview plus:
- Execution status per resource
- Success/failure indicators
- Error messages

## Resource Status

| Status | Display |
|--------|---------|
| Pending | Dimmed |
| Running | Spinner |
| Success | Green checkmark |
| Failed | Red X |

## Cancellation

Press `Esc` during execution to cancel. Note: Some operations may not be cancellable mid-execution.

## Related

- [Preview](preview.md) - Preview before executing
- [Resource Targeting](resource-targetting.md) - Target specific resources
