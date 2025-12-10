# Copy

![Copy Demo](../assets/copy.gif)

Copy resource data to clipboard.

## Keys

| Key | Action |
|-----|--------|
| `y` | Copy selected resource as JSON |
| `Y` | Copy all visible resources as JSON |

## JSON Format

```json
{
  "urn": "urn:pulumi:dev::project::type::name",
  "type": "provider:module/resource:Resource",
  "inputs": {
    "key": "value"
  },
  "outputs": {
    "key": "value"
  }
}
```

For multiple resources, outputs array of objects.

## Clipboard Commands

Platform-specific clipboard access:
- **macOS**: `pbcopy`
- **Linux**: `xclip` or `xsel`
- **Windows**: `clip`

## Feedback

On successful copy:
- Toast notification shows count
- Flash animation highlights copied items

## Use Cases

- Export resource data for documentation
- Share resource configuration
- Debug by inspecting full resource state
- Transfer URNs to other tools

## Implementation

- `internal/ui/clipboard.go` - Clipboard access
- `internal/ui/resourcecopy.go` - JSON serialization
- `cmd/p5/logic.go` - `FormatClipboardMessage()`
