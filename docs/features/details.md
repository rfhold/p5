# Details Panel

![Details Demo](../assets/details.gif)

View detailed information about selected resources.

## Toggle

Press `D` to toggle the details panel.

## Content

### Stack View
Shows resource outputs from Pulumi state:
- URN
- Type
- All output properties

### Preview View
Shows diff between current and proposed state:
- Added properties (green `+`)
- Removed properties (red `-`)
- Changed properties (yellow `~`)

### History View
Shows update details for selected history entry:
- Version and operation type
- Start/end times
- Result status
- Resource changes summary

## Navigation

When details panel is focused (`FocusDetailsPanel`):
- `j`/`k` or arrows: Scroll content
- `PgUp`/`PgDn`: Page scroll
- `g`/`G`: Jump to top/bottom
- `Esc` or `D`: Close panel

## Layout

Details panel appears on the right side of the screen. Width is proportional to terminal width.

## Implementation

- `internal/ui/details.go` - Main details panel
- `internal/ui/historydetails.go` - History-specific details
- `internal/ui/diff.go` - Diff rendering logic
