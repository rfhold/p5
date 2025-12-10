# Cloudflare Plugin

Builtin plugin for Cloudflare resources.

## Capabilities

- **Import Helper**: Provides import suggestions for Cloudflare resources

## Status

Currently a stub implementation - returns empty suggestions for `cloudflare:*` resources.

## Configuration

```yaml
# Pulumi.yaml
p5:
  plugins:
    cloudflare:
      import_helper: true
```

## Supported Resources

Matches all `cloudflare:*` resource types (stub implementation).

## Implementation

Located in `internal/plugins/builtins/cloudflare.go`.
