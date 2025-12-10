# Plugin Interface

Plugins extend p5 with authentication, import suggestions, and resource opening capabilities.

## Package

```go
import "github.com/rfhold/p5/pkg/plugin"
```

## Interfaces

### AuthPlugin (Required)

Every plugin must implement authentication:

```go
type AuthPlugin interface {
    Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error)
}
```

**Request:**
- `ProgramConfig` - Configuration from `Pulumi.yaml`
- `StackConfig` - Configuration from `Pulumi.{stack}.yaml`
- `StackName`, `ProgramName` - Current identifiers
- `SecretsProvider` - Stack secrets provider

**Response:**
- `Success` - Authentication result
- `Env` - Environment variables to set
- `TtlSeconds` - Credential lifetime (0=never expires, -1=always re-auth)
- `Error` - Error message if failed

### ImportHelperPlugin (Optional)

Provides import ID suggestions:

```go
type ImportHelperPlugin interface {
    GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) (*ImportSuggestionsResponse, error)
}
```

### ResourceOpenerPlugin (Optional)

Opens resources in external tools:

```go
type ResourceOpenerPlugin interface {
    GetSupportedOpenTypes(ctx context.Context, req *SupportedOpenTypesRequest) (*SupportedOpenTypesResponse, error)
    OpenResource(ctx context.Context, req *OpenResourceRequest) (*OpenResourceResponse, error)
}
```

Open actions:
- **Browser**: Opens URL in default browser
- **Exec**: Launches alternate screen program (e.g., k9s)

## Configuration

### Sources

1. **Global** (`p5.toml`): TOML at git root, defaults for all programs
2. **Program** (`Pulumi.yaml`): YAML under `p5:` key
3. **Stack** (`Pulumi.{stack}.yaml`): Per-stack under `config: -> p5:plugins`

### PluginConfig

```go
type PluginConfig struct {
    Cmd            string           // External plugin executable
    Args           []string         // Command arguments
    Config         map[string]any   // Plugin-specific config
    Refresh        *RefreshTrigger  // When to refresh credentials
    ImportHelper   bool             // Enable import helper
    UseAuthEnv     bool             // Pass auth env to import/opener
    ResourceOpener bool             // Enable resource opener
}
```

### RefreshTrigger

```go
type RefreshTrigger struct {
    OnWorkspaceChange *bool  // Default: true
    OnStackChange     *bool  // Default: true
    OnConfigChange    *bool  // Default: false
}
```

## External Plugins

External plugins use gRPC via HashiCorp's `go-plugin`:

```go
package main

import "github.com/rfhold/p5/pkg/plugin"

type MyPlugin struct{}

func (p *MyPlugin) Authenticate(ctx context.Context, req *plugin.AuthenticateRequest) (*plugin.AuthenticateResponse, error) {
    env := map[string]string{"MY_TOKEN": "secret"}
    return plugin.SuccessResponse(env, 3600), nil // 1 hour TTL
}

func main() {
    plugin.Serve(&MyPlugin{})
}
```

Configuration:
```yaml
p5:
  plugins:
    my-plugin:
      cmd: /path/to/my-plugin
      args: ["--verbose"]
```

## Authentication Flow

1. Plugins in `order` array run sequentially (credentials cached for subsequent plugins)
2. Remaining plugins run in parallel
3. Credentials cached with TTL tracking
4. Re-authentication on workspace/stack change (configurable)

## Credential Lifecycle

- `TTL > 0`: Expires after specified seconds
- `TTL = 0`: Never expires
- `TTL = -1`: Always re-authenticate
