# p5 Plugin System

The p5 plugin system provides authentication credentials for Pulumi stacks. Plugins are triggered when a stack or workspace is selected, and can set environment variables used by Pulumi operations.

p5 supports two types of plugins:
- **Builtin plugins**: Run in-process, shipped with p5 (e.g., `env`)
- **External plugins**: Separate Go binaries using [go-plugin](https://github.com/hashicorp/go-plugin) over gRPC

## Overview

```
┌─────────────────────────────────────────────────────────┐
│  p5 TUI                                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Plugin Manager                                    │  │
│  │  - Loads plugins on workspace/stack change        │  │
│  │  - Runs authentication in parallel                │  │
│  │  - Caches credentials with TTL                    │  │
│  └───────────────────────────────────────────────────┘  │
│                          │                              │
│          ┌───────────────┼───────────────┐              │
│          ▼               ▼               ▼              │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐    │
│  │ env          │ │ okta-aws     │ │ gcp-auth     │    │
│  │ (builtin)    │ │ (external)   │ │ (external)   │    │
│  └──────────────┘ └──────────────┘ └──────────────┘    │
│          │               │               │              │
│          ▼               ▼               ▼              │
│   PULUMI_BACKEND  AWS_ACCESS_KEY   GOOGLE_CREDS        │
│   PULUMI_PASS...  AWS_SECRET_...                       │
└─────────────────────────────────────────────────────────┘
```

## Builtin Plugins

Builtin plugins are compiled into p5 and run in-process without spawning subprocesses.

### env Plugin

The `env` plugin loads environment variables from multiple sources.

#### Source Types

| Type | Description |
|------|-------------|
| `file` | Load from a `.env` file |
| `static` | Define env vars directly in config |
| `exec` | Run a command that outputs `.env` format to stdout |

#### Simple Configuration

For a single source, configure directly:

```toml
# File source (type inferred from "path")
[plugins.env.config]
path = ".env"

# Static source
[plugins.env.config]
type = "static"
vars = '{"PULUMI_BACKEND_URL": "file://~/.pulumi", "PULUMI_CONFIG_PASSPHRASE": ""}'

# Exec source
[plugins.env.config]
cmd = "./scripts/get-secrets.sh"
args = '["--format", "env"]'
dir = "/path/to/workdir"  # optional
```

#### Multiple Sources

Mix and match sources using the `sources` array. Sources are processed in order, with later sources overriding earlier ones:

```toml
[plugins.env.config]
sources = '''
[
  {"type": "file", "path": "~/.env.shared"},
  {"type": "static", "vars": {"APP_ENV": "development"}},
  {"type": "exec", "cmd": "vault", "args": ["kv", "get", "-format=env", "secret/app"]}
]
'''
```

#### Practical Example

```toml
[plugins.env.config]
sources = '''
[
  {"type": "file", "path": ".env.defaults"},
  {"type": "file", "path": ".env.local"},
  {"type": "exec", "cmd": "./scripts/get-secrets.sh"}
]
'''
```

This loads:
1. Default values from `.env.defaults`
2. Local overrides from `.env.local`
3. Secrets from a script (highest priority)

## Configuration

Plugin configuration can be defined in multiple locations, with more specific configs taking precedence.

### p5.toml (Global Config)

The `p5.toml` file provides global plugin configuration for an entire repository or directory tree. p5 looks for this file in:
1. Git repository root (if in a git repo)
2. Directory where p5 was launched

```toml
[plugins.env]
[plugins.env.config]
type = "static"
vars = '{"PULUMI_BACKEND_URL": "file://~/.pulumi"}'

[plugins.okta-aws]
source = "./plugins/okta-aws"

[plugins.okta-aws.config]
clientId = "0oa1234567890abcdef"
orgUrl = "https://mycompany.okta.com"

# Control when credentials are refreshed
[plugins.okta-aws.refresh]
onWorkspaceChange = true   # default: true
onStackChange = true       # default: true
onConfigChange = false     # default: false
```

### Pulumi.yaml (Program Level)

Program-specific configuration that overrides global settings:

```yaml
name: my-infra
runtime: go

p5:
  plugins:
    okta-aws:
      source: "./plugins/okta-aws"
      config:
        clientId: "0oa1234567890abcdef"
        orgUrl: "https://mycompany.okta.com"
      refresh:
        onWorkspaceChange: true
        onStackChange: true
        onConfigChange: false
```

### Pulumi.{stack}.yaml (Stack Level)

Stack-specific configuration:

```yaml
config:
  aws:region: us-west-2
  
  # Plugin config for this specific stack
  p5:plugins:
    okta-aws:
      config:
        awsAccountId: "123456789012"
        roleArn: "arn:aws:iam::123456789012:role/PulumiRole"
```

### Configuration Precedence

Configuration is merged with the following precedence (highest to lowest):
1. **Stack config** (`Pulumi.{stack}.yaml`) - stack-specific values
2. **Program config** (`Pulumi.yaml`) - program-specific overrides
3. **Global config** (`p5.toml`) - shared defaults

For plugin settings:
- `source` from higher-precedence config completely overrides lower
- `config` maps are merged (higher-precedence values override same keys)
- `refresh` settings from higher-precedence config completely overrides lower

## Plugin Sources

| Source Type | Example | Description |
|-------------|---------|-------------|
| Builtin | `env` | No source needed, name-only |
| Local Path | `./plugins/my-plugin` | Path to plugin binary |
| Absolute Path | `/usr/local/bin/p5-okta-aws` | Absolute path to binary |

## External Plugin Development

External plugins are Go binaries that communicate with p5 via gRPC using HashiCorp's go-plugin library.

### Plugin Interface

Plugins must implement the `AuthPlugin` interface:

```go
type AuthPlugin interface {
    Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error)
}
```

### Request/Response Types

**AuthenticateRequest:**
```go
type AuthenticateRequest struct {
    ProgramConfig map[string]string // Config from Pulumi.yaml
    StackConfig   map[string]string // Config from Pulumi.{stack}.yaml
    StackName     string            // Current stack name
    ProgramName   string            // Current program name
}
```

**AuthenticateResponse:**
```go
type AuthenticateResponse struct {
    Success    bool              // true if authentication succeeded
    Env        map[string]string // Environment variables to set
    TtlSeconds int32             // Credential TTL (see below)
    Error      string            // Error message if Success is false
}
```

**TTL values:**
- `> 0`: Credentials expire after this many seconds
- `0`: Credentials never expire (reload on stack/workspace change)
- `-1`: Always re-authenticate (plugin handles caching internally)

### Example External Plugin

```go
package main

import (
    "context"
    "fmt"

    "github.com/rfhold/p5/pkg/plugin"
)

type OktaAWSPlugin struct{}

func (p *OktaAWSPlugin) Authenticate(ctx context.Context, req *plugin.AuthenticateRequest) (*plugin.AuthenticateResponse, error) {
    // Extract config
    clientID := req.ProgramConfig["clientId"]
    orgURL := req.ProgramConfig["orgUrl"]
    awsAccountID := req.StackConfig["awsAccountId"]
    roleArn := req.StackConfig["roleArn"]

    if clientID == "" || orgURL == "" {
        return plugin.NewErrorResponse("missing required config: clientId, orgUrl"), nil
    }
    if awsAccountID == "" || roleArn == "" {
        return plugin.NewErrorResponse("missing required stack config: awsAccountId, roleArn"), nil
    }

    // Perform authentication (OAuth flow, STS assume role, etc.)
    creds, err := performOktaAuth(ctx, clientID, orgURL, awsAccountID, roleArn)
    if err != nil {
        return plugin.NewErrorResponse(fmt.Sprintf("auth failed: %v", err)), nil
    }

    return plugin.NewSuccessResponse(map[string]string{
        "AWS_ACCESS_KEY_ID":     creds.AccessKeyID,
        "AWS_SECRET_ACCESS_KEY": creds.SecretAccessKey,
        "AWS_SESSION_TOKEN":     creds.SessionToken,
    }, 3600), nil  // 1 hour TTL
}

func main() {
    plugin.Serve(&OktaAWSPlugin{})
}
```

### Building

```bash
go build -o okta-aws ./cmd/okta-aws
```

Then reference it in your config:

```toml
[plugins.okta-aws]
source = "./okta-aws"
```

### pkg/plugin Package

The `pkg/plugin` package provides everything external plugin authors need:

| Export | Description |
|--------|-------------|
| `AuthPlugin` | Interface to implement |
| `AuthenticateRequest` | Request type |
| `AuthenticateResponse` | Response type |
| `Serve(impl)` | Start the plugin server |
| `NewSuccessResponse(env, ttl)` | Create success response |
| `NewErrorResponse(err)` | Create error response |
| `Handshake` | Handshake config (for advanced use) |

## Lifecycle

1. **Workspace Selection**: Plugin manager checks refresh triggers for each plugin
   - If `onWorkspaceChange: true` (default), credentials are invalidated
   - If `onConfigChange: true`, only invalidates if plugin config actually changed
2. **Stack Selection**: Plugin manager checks refresh triggers for each plugin
   - If `onStackChange: true` (default), credentials are invalidated
   - If `onConfigChange: true`, only invalidates if plugin config actually changed
3. **Authentication**: All plugins run in parallel
4. **Credential Caching**: Successful credentials cached with TTL, plus config hash for change detection
5. **Operation**: Cached env vars passed to Pulumi operations
6. **Refresh**: On TTL expiry or `ttl_seconds: -1`, re-authenticate before next operation

### Refresh Trigger Options

| Setting | Default | Description |
|---------|---------|-------------|
| `onWorkspaceChange` | `true` | Refresh credentials when workspace changes |
| `onStackChange` | `true` | Refresh credentials when stack changes |
| `onConfigChange` | `false` | Only refresh if plugin config (program + stack) actually changed |

**Example: Same credentials across all stacks in a workspace**

```toml
[plugins.my-auth]
source = "./my-auth"
[plugins.my-auth.refresh]
onStackChange = false  # Don't refresh when switching stacks
```

**Example: Only refresh when config actually differs**

```toml
[plugins.okta-aws]
source = "./okta-aws"
[plugins.okta-aws.refresh]
onWorkspaceChange = true
onStackChange = true
onConfigChange = true  # Only refresh if config actually changed between stacks
```

## Troubleshooting

### Plugin Load Errors

Check that:
- For builtin plugins: The plugin name is correct (e.g., `env`)
- For external plugins: The binary exists at the specified source path
- The binary is executable and built for your platform

### Authentication Failures

- Check toast messages in the TUI for error details
- Verify program config in `Pulumi.yaml`
- Verify stack config in `Pulumi.{stack}.yaml`
- For external plugins, test the binary directly

### External Plugin Debugging

Run the plugin binary directly to check for build/runtime errors:

```bash
./plugins/my-plugin
# Should output: "This binary is a plugin..."
```

If it crashes or errors, fix the plugin code.
