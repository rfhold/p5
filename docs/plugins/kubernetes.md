# Kubernetes Plugin

Builtin plugin for Kubernetes import suggestions.

## Capabilities

- **Import Helper**: Suggests import IDs by querying kubectl

## Configuration

```yaml
# Pulumi.yaml
p5:
  plugins:
    kubernetes:
      import_helper: true
      use_auth_env: true
```

## Behavior

Runs `kubectl get <resource> -o json` to list existing resources and returns suggestions.

## Supported Resources

All `kubernetes:*` resource types.

### Cluster-Scoped Kinds

Resources without namespace:
- Namespace
- Node
- PersistentVolume
- ClusterRole
- ClusterRoleBinding
- StorageClass
- IngressClass
- ClusterIssuer

### Namespaced Kinds

Returns suggestions in `namespace/name` format.

## Kubeconfig Handling

Sources (in order):
1. Provider inputs (`kubeconfig` key)
2. Stack config
3. Program config

Kubeconfig can be:
- File path - Used directly via `--kubeconfig`
- YAML content - Written to temp file

## Import ID Format

| Resource Type | Format |
|--------------|--------|
| Cluster-scoped | `name` |
| Namespaced | `namespace/name` |

## Implementation

Located in `internal/plugins/builtins/kubernetes.go`.
