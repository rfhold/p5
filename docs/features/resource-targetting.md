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

## Selection Modes

### Discrete Selection (Space)

Press `Space` to toggle selection on individual resources. Discrete selections are independent and persist until explicitly cleared.

1. Navigate to a resource
2. Press `Space` to toggle its selection
3. Navigate and select additional resources as needed
4. Press flag key to apply to all selected
5. Press `Esc` to clear all discrete selections

Discrete selections persist after applying flags, allowing you to apply multiple flag operations to the same set of resources.

### Visual Mode (v)

Press `v` to enter visual selection mode for selecting a contiguous range.

1. Press `v` to start selection at cursor
2. Navigate to extend selection range
3. Press flag key to apply to all in range
4. Press `Esc` to exit visual mode

Visual mode exits automatically after applying flags.

### Combined Selection

Both selection modes can be used together:

- Use `Space` inside visual mode to toggle discrete selection on the entire visual range
- Flag operations apply to the union of discrete selections and visual range
- Discrete selections persist independently of visual mode

Selection highlighting uses distinct colors:
- **Blue** - Visual range only
- **Green** - Discrete selection only  
- **Purple** - Both visual and discrete selection

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
