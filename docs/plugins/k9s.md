# k9s Plugin

Builtin plugin for opening Kubernetes resources in k9s TUI.

## Capabilities

- **Resource Opener**: Launches k9s for Kubernetes resources

## Configuration

```yaml
# Pulumi.yaml
p5:
  plugins:
    k9s:
      resource_opener: true
      use_auth_env: true  # Pass auth env vars to k9s
```

## Supported Resources

All `kubernetes:*` resource types.

## Behavior

Launches k9s with:
- `--context` - Kubernetes context from provider
- `--namespace` - Resource namespace
- `--kubeconfig` - Kubeconfig file (writes temp file if content provided)
- `--command <kind>` - Navigates to resource type

## Kubeconfig Handling

Kubeconfig can be provided as:
1. File path - Used directly
2. YAML content - Written to temp file

Sources (in order):
1. Provider inputs
2. Stack config
3. Program config

## Usage

1. Enable resource opener in config
2. Navigate to a Kubernetes resource in p5
3. Press `o` to launch k9s

k9s opens in alternate screen mode, returning to p5 on exit.

## Implementation

Located in `internal/plugins/builtins/k9s.go`.
