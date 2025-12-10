# Import

Import existing cloud resources into Pulumi state.

![Import Resource](../assets/import.gif)

## Access

Only available in preview view for resources with `Create` operation.

| Key | Action |
|-----|--------|
| `I` | Open import modal for selected resource |

## Modal

Import modal shows:
- Resource type and name
- Plugin suggestions (if available)
- Text input for import ID

## Plugin Suggestions

Plugins implementing `ImportHelperPlugin` can provide suggestions:
- **kubernetes**: Lists resources via `kubectl get`
- **cloudflare**: Stub implementation

Suggestions are fetched asynchronously when modal opens.

## Flow

1. Run preview (`u`) to see create operations
2. Select a resource with `+` (create) operation
3. Press `I` to open import modal
4. Select suggestion or enter import ID manually
5. Confirm to execute import
6. On success: toast notification, re-runs preview
7. On failure: error modal with details

## Import ID Format

Format varies by provider:
- AWS: Resource ARN or ID
- Kubernetes: `namespace/name` or `name`
- GCP: Resource self-link or ID

## Validation

`CanImportResource()` checks:
- Currently in preview view
- Selected item has create operation
- Valid resource item selected

## Implementation

- `cmd/p5/commands.go` - `showImportModal()`, `executeImport()`
- `internal/ui/importmodal.go` - Import modal component
- `internal/pulumi/import.go` - Import execution
