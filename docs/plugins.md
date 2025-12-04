# p5 Plugin System

The p5 plugin system allows WASM plugins to provide authentication credentials for Pulumi stacks. Plugins are triggered when a stack or workspace is selected, and can set environment variables used by Pulumi operations.

## Overview

```
┌─────────────────────────────────────────────────────────┐
│  p5 TUI                                                  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Plugin Manager                                    │  │
│  │  - Loads plugins on workspace/stack change        │  │
│  │  - Runs authentication in parallel                │  │
│  │  - Caches credentials with TTL                    │  │
│  └───────────────────────────────────────────────────┘  │
│                          │                               │
│              ┌───────────┴───────────┐                  │
│              ▼                       ▼                  │
│      ┌──────────────┐       ┌──────────────┐           │
│      │ Plugin A     │       │ Plugin B     │           │
│      │ (okta-aws)   │       │ (gcp-auth)   │           │
│      └──────────────┘       └──────────────┘           │
│              │                       │                  │
│              ▼                       ▼                  │
│      AWS_ACCESS_KEY_ID       GOOGLE_APPLICATION_...    │
│      AWS_SECRET_ACCESS_KEY                              │
│      AWS_SESSION_TOKEN                                  │
└─────────────────────────────────────────────────────────┘
```

## Configuration

Plugin configuration can be defined in multiple locations, with more specific configs taking precedence.

### p5.toml (Global Config)

The `p5.toml` file provides global plugin configuration for an entire repository or directory tree. p5 looks for this file in:
1. Git repository root (if in a git repo)
2. Directory where p5 was launched

This is useful for defining shared plugin settings across multiple Pulumi programs.

```toml
[plugins.okta-aws]
source = "github.com/myorg/p5-plugins/okta-aws@v1.0.0"

[plugins.okta-aws.config]
clientId = "0oa1234567890abcdef"
orgUrl = "https://mycompany.okta.com"

# Control when credentials are refreshed
[plugins.okta-aws.refresh]
onWorkspaceChange = true   # default: true
onStackChange = true       # default: true
onConfigChange = false     # default: false - when true, only refresh if config actually changed
```

### Pulumi.yaml (Program Level)

Program-specific configuration that overrides global settings:

```yaml
name: my-infra
runtime: go

p5:
  plugins:
    okta-aws:
      source: "github.com/myorg/p5-plugins/okta-aws@v1.0.0"
      config:
        clientId: "0oa1234567890abcdef"
        orgUrl: "https://mycompany.okta.com"
      # Optional: control refresh behavior per-program
      refresh:
        onWorkspaceChange: true
        onStackChange: true
        onConfigChange: false
```

### Pulumi.{stack}.yaml (Stack Level)

Stack-specific configuration (usually contains AWS account IDs, role ARNs, etc.):

```yaml
config:
  aws:region: us-west-2
  
  # Plugin config for this specific stack
  p5:plugins:okta-aws:
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

Plugins can be loaded from multiple sources:

| Source Type | Example |
|-------------|---------|
| GitHub Release | `github.com/myorg/repo/plugin-name@v1.0.0` |
| HTTP URL | `https://example.com/plugin.wasm` |
| Local Path | `file://./plugins/my-plugin` or `./plugins/my-plugin` |

Plugins are cached in `~/.p5/plugins/` by version. When the version changes, the new version is downloaded.

## Plugin Manifest

Each plugin must include a manifest file (`{name}.manifest.yaml`) alongside the `.wasm` file:

```yaml
name: okta-aws
version: "1.0.0"
description: "Authenticate with AWS via Okta OIDC"

# Security permissions (deny-by-default)
permissions:
  # HTTP hosts the plugin can access (supports wildcards)
  http:
    - "*.okta.com"
    - "sts.amazonaws.com"
    - "sts.*.amazonaws.com"
  
  # Environment variables the plugin can set
  env:
    - "AWS_ACCESS_KEY_ID"
    - "AWS_SECRET_ACCESS_KEY"
    - "AWS_SESSION_TOKEN"

# Allow interactive browser-based auth
interactive: true

# Config schema (for documentation)
config:
  program:
    clientId:
      type: string
      required: true
      description: "Okta OAuth Client ID"
    orgUrl:
      type: string
      required: true
      description: "Okta organization URL"
  stack:
    awsAccountId:
      type: string
      required: true
    roleArn:
      type: string
      required: true
```

## Plugin API

### Exported Function

Plugins must export an `authenticate` function:

```
authenticate(input: JSON) -> JSON
```

**Input:**
```json
{
  "program_config": {
    "clientId": "0oa1234...",
    "orgUrl": "https://mycompany.okta.com"
  },
  "stack_config": {
    "awsAccountId": "123456789012",
    "roleArn": "arn:aws:iam::..."
  },
  "stack_name": "dev",
  "program_name": "my-infra"
}
```

**Output:**
```json
{
  "success": true,
  "env": {
    "AWS_ACCESS_KEY_ID": "ASIA...",
    "AWS_SECRET_ACCESS_KEY": "...",
    "AWS_SESSION_TOKEN": "..."
  },
  "ttl_seconds": 3600,
  "error": null
}
```

**TTL values:**
- `> 0`: Credentials expire after this many seconds
- `0`: Credentials never expire (use for static credentials)
- `-1`: Always re-authenticate (plugin handles caching internally)

### Host Functions

The host provides these functions to plugins:

#### `http_request`

Make HTTP requests to allowed hosts.

```go
//go:wasmimport extism:host/user http_request
func httpRequest(ptr uint64) uint64
```

**Input:**
```json
{
  "method": "POST",
  "url": "https://mycompany.okta.com/oauth2/v1/token",
  "headers": {
    "Content-Type": "application/x-www-form-urlencoded"
  },
  "body": "grant_type=authorization_code&code=..."
}
```

**Output:**
```json
{
  "status_code": 200,
  "headers": {"Content-Type": "application/json"},
  "body": "{...}",
  "error": null
}
```

#### `open_browser`

Open a URL in the user's default browser (requires `interactive: true` in manifest).

```go
//go:wasmimport extism:host/user open_browser
func openBrowser(ptr uint64) uint64
```

**Input:**
```json
{"url": "https://mycompany.okta.com/oauth2/authorize?..."}
```

#### `wait_for_callback`

Start a local HTTP server and wait for an OAuth callback.

```go
//go:wasmimport extism:host/user wait_for_callback
func waitForCallback(ptr uint64) uint64
```

**Input:**
```json
{
  "port": 8765,
  "path": "/callback",
  "timeout_seconds": 120
}
```

**Output:**
```json
{
  "query_params": {
    "code": "auth_code_here",
    "state": "..."
  },
  "error": null
}
```

## Example Plugin (TinyGo)

### main.go

```go
package main

import (
	"encoding/json"

	"github.com/extism/go-pdk"
)

// Host function imports
//go:wasmimport extism:host/user http_request
func hostHTTPRequest(ptr uint64) uint64

//go:wasmimport extism:host/user open_browser
func hostOpenBrowser(ptr uint64) uint64

//go:wasmimport extism:host/user wait_for_callback
func hostWaitForCallback(ptr uint64) uint64

type AuthInput struct {
	ProgramConfig map[string]interface{} `json:"program_config"`
	StackConfig   map[string]interface{} `json:"stack_config"`
	StackName     string                 `json:"stack_name"`
	ProgramName   string                 `json:"program_name"`
}

type AuthOutput struct {
	Success    bool              `json:"success"`
	Env        map[string]string `json:"env,omitempty"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Error      string            `json:"error,omitempty"`
}

//go:wasmexport authenticate
func authenticate() int32 {
	var input AuthInput
	if err := pdk.InputJSON(&input); err != nil {
		outputError("failed to parse input: " + err.Error())
		return 1
	}

	// Extract config
	clientID, _ := input.ProgramConfig["clientId"].(string)
	orgURL, _ := input.ProgramConfig["orgUrl"].(string)
	awsAccountID, _ := input.StackConfig["awsAccountId"].(string)
	roleArn, _ := input.StackConfig["roleArn"].(string)

	if clientID == "" || orgURL == "" {
		outputError("missing required program config: clientId, orgUrl")
		return 1
	}
	if awsAccountID == "" || roleArn == "" {
		outputError("missing required stack config: awsAccountId, roleArn")
		return 1
	}

	// 1. Start OAuth flow
	authURL := orgURL + "/oauth2/v1/authorize?client_id=" + clientID +
		"&response_type=code&scope=openid&redirect_uri=http://localhost:8765/callback"

	if err := openBrowser(authURL); err != nil {
		outputError("failed to open browser: " + err.Error())
		return 1
	}

	// 2. Wait for callback
	params, err := waitForCallback(8765, "/callback", 120)
	if err != nil {
		outputError("failed to get callback: " + err.Error())
		return 1
	}

	code := params["code"]
	if code == "" {
		outputError("no authorization code received")
		return 1
	}

	// 3. Exchange code for tokens
	tokens, err := exchangeCodeForTokens(orgURL, clientID, code)
	if err != nil {
		outputError("token exchange failed: " + err.Error())
		return 1
	}

	// 4. Assume AWS role with OIDC token
	creds, err := assumeAWSRole(tokens.IDToken, roleArn, awsAccountID)
	if err != nil {
		outputError("AWS assume role failed: " + err.Error())
		return 1
	}

	output := AuthOutput{
		Success: true,
		Env: map[string]string{
			"AWS_ACCESS_KEY_ID":     creds.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY": creds.SecretAccessKey,
			"AWS_SESSION_TOKEN":     creds.SessionToken,
		},
		TTLSeconds: 3600,
	}

	pdk.OutputJSON(output)
	return 0
}

func outputError(msg string) {
	output := AuthOutput{
		Success: false,
		Error:   msg,
	}
	pdk.OutputJSON(output)
}

// Helper functions for HTTP, browser, callback...
// (implementation details omitted for brevity)

func main() {}
```

### Building

```bash
tinygo build -o okta-aws.wasm -target wasip1 -buildmode=c-shared main.go
```

### Manifest (okta-aws.manifest.yaml)

```yaml
name: okta-aws
version: "1.0.0"
description: "AWS credentials via Okta OIDC"

permissions:
  http:
    - "*.okta.com"
    - "sts.amazonaws.com"
    - "sts.*.amazonaws.com"
  env:
    - "AWS_ACCESS_KEY_ID"
    - "AWS_SECRET_ACCESS_KEY"
    - "AWS_SESSION_TOKEN"

interactive: true

config:
  program:
    clientId:
      type: string
      required: true
    orgUrl:
      type: string
      required: true
  stack:
    awsAccountId:
      type: string
      required: true
    roleArn:
      type: string
      required: true
```

## Security Model

The plugin system follows a **deny-by-default** security model:

| Capability | Default | Controlled By |
|------------|---------|---------------|
| HTTP requests | Denied | `permissions.http` in manifest |
| Set env vars | Denied | `permissions.env` in manifest |
| Open browser | Denied | `interactive: true` in manifest |
| Filesystem | Denied | Not available |

**Enforcement:**
- HTTP requests to disallowed hosts return an error
- Setting disallowed env vars returns an error and fails authentication
- Browser/callback functions fail if `interactive: false`

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

If your plugin provides credentials that are workspace-level (not stack-specific), you can disable stack-based refresh:

```toml
[plugins.my-auth]
source = "..."
[plugins.my-auth.refresh]
onStackChange = false  # Don't refresh when switching stacks
```

**Example: Only refresh when config actually differs**

If multiple stacks share the same plugin configuration, you can avoid unnecessary re-authentication:

```toml
[plugins.okta-aws]
source = "..."
[plugins.okta-aws.refresh]
onWorkspaceChange = true
onStackChange = true
onConfigChange = true  # Only refresh if config actually changed between stacks
```

## Troubleshooting

### Plugin Load Errors

Check that:
- The `.wasm` and `.manifest.yaml` files exist at the source
- The manifest is valid YAML
- Required fields (`name`, `version`) are present

### Authentication Failures

- Check toast messages in the TUI for error details
- Verify program config in `Pulumi.yaml`
- Verify stack config in `Pulumi.{stack}.yaml`
- Check that HTTP hosts in manifest match what the plugin accesses

### Permission Errors

If you see "host not allowed" or "env var not allowed":
- Update the manifest's `permissions.http` or `permissions.env`
- Rebuild and redeploy the plugin
