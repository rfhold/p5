# Authentication

Plugin-based authentication for cloud providers and tools.

## Flow

1. Plugins load during initialization
2. Authentication runs for each plugin
3. Credentials cached with TTL
4. Environment variables set for Pulumi operations

## Plugin Order

Plugins in `order` array authenticate sequentially. Credentials from earlier plugins are available to later ones.

```yaml
p5:
  plugins:
    order: [env, aws]  # env runs first, aws can use its env vars
    env:
      config:
        path: .env
    aws:
      cmd: /path/to/aws-plugin
```

Plugins not in `order` run in parallel after ordered plugins complete.

## Credential Caching

| TTL | Behavior |
|-----|----------|
| `> 0` | Expires after N seconds |
| `= 0` | Never expires |
| `= -1` | Always re-authenticate |

## Refresh Triggers

Configure when credentials refresh:

```yaml
p5:
  plugins:
    my-plugin:
      refresh:
        on_workspace_change: true   # Default: true
        on_stack_change: true       # Default: true
        on_config_change: false     # Default: false
```

## Busy Lock

During authentication, app shows "Authenticating..." status. Operations queue and execute after auth completes.

## Environment Variables

Authenticated credentials are merged and passed to:
- Pulumi Automation API operations
- Import helper plugin requests (if `use_auth_env: true`)
- Resource opener plugin requests (if `use_auth_env: true`)

## Implementation

- `internal/plugins/auth.go` - Authentication logic
- `internal/plugins/manager.go` - Credential management
- `cmd/p5/update_init.go` - Init-time authentication
