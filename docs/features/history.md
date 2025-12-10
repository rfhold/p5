# History

![History Demo](../assets/history.gif)

Browse stack update history.

## Access

Press `h` to switch to history view.

## Display

Shows list of previous updates:
- Version number
- Operation type (up/destroy/refresh)
- Start time
- Duration
- Result (succeeded/failed)
- User info

## Navigation

- `j`/`k` or arrows: Move selection
- `Enter`: View details (with `D` panel open)
- `Esc`: Return to stack view

## Details

With details panel open (`D`), selected history entry shows:
- Full update metadata
- Resource changes summary
- Create/update/delete counts

## Data Source

History is fetched via `StackReader.GetHistory()` which calls `stack.History()` from the Pulumi Automation API.

## Pagination

History is fetched with pagination support:
- Default page size: 20 entries
- Additional pages loaded on scroll (if implemented)

## Implementation

- `cmd/p5/update_operations.go` - `handleStackHistory()`
- `internal/ui/historylist.go` - History list component
- `internal/ui/historydetails.go` - History details component
