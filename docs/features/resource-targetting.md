# Resource Targeting

![Targeting Demo](../assets/targeting.gif)

Mark resources for targeted operations.

## Flags

| Key | Flag | Pulumi Equivalent | Description |
|-----|------|-------------------|-------------|
| `T` | Target | `--target` | Only operate on these resources |
| `R` | Replace | `--replace` | Force replacement |
| `E` | Exclude | `--exclude` | Exclude from operation |

## Behavior

- **Target**: Only flagged resources are included in operation
- **Replace**: Flagged resources are replaced instead of updated
- **Exclude**: Flagged resources are skipped

Target and Replace are mutually exclusive with Exclude. Setting one clears the other.

## Visual Mode

Press `v` to enter visual selection mode for bulk operations.

1. Press `v` to start selection
2. Navigate to extend selection
3. Press flag key to apply to all selected
4. Press `Esc` to exit visual mode

## Clear Flags

| Key | Action |
|-----|--------|
| `c` | Clear flags on current resource |
| `C` | Clear all flags |

## Display

Flagged resources show indicators:
- `[T]` - Target
- `[R]` - Replace
- `[X]` - Exclude

## Persistence

Flags persist across:
- View switches (stack → preview → execute)
- Preview refreshes

Flags reset on:
- Stack change
- Application restart

## Implementation

- `internal/ui/resourceflags.go` - Flag types and display
- `cmd/p5/state.go` - Flag storage in `AppState.Flags`
- `cmd/p5/update_keys.go` - Flag toggle handlers
