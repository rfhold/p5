# Migration Plan: Extism/WASM to HashiCorp go-plugin

## Overview

Migrate the p5 plugin system from Extism/WASM to HashiCorp go-plugin. This simplifies plugin development (native Go instead of TinyGo/WASM), removes the manifest security model (which wasn't truly enforceable), and aligns with the broader Go infrastructure ecosystem.

## Key Changes

| Aspect | Current (Extism) | New (go-plugin) |
|--------|------------------|-----------------|
| Plugin format | `.wasm` file + manifest | Native binary (any executable) |
| Plugin source | Git repo, HTTP URL, local path | Command + args |
| Security model | Manifest-declared permissions | Trust-based (user responsibility) |
| Build toolchain | TinyGo + special flags | Standard `go build` |
| Communication | WASM host functions | gRPC over local socket |
| Plugin language | TinyGo, Rust, etc. | Go (or any gRPC-capable language) |

## Plugin Configuration

### Before (p5.toml / Pulumi.yaml)

```toml
[plugins.dev-env]
source = "./plugins/dev-env"  # or github.com/org/repo@version

[plugins.dev-env.config]
backendUrl = "file://~/.pulumi"
```

### After

```toml
[plugins.dev-env]
cmd = "./plugins/dev-env/dev-env"
args = []  # optional

[plugins.dev-env.config]
backendUrl = "file://~/.pulumi"
```

Or with arguments:

```toml
[plugins.okta-aws]
cmd = "/usr/local/bin/p5-plugin-okta"
args = ["--verbose", "--region", "us-west-2"]

[plugins.okta-aws.config]
clientId = "0oa1234567890abcdef"
```

## gRPC Interface

Single service with `Authenticate` RPC, maintaining parity with current functionality:

```protobuf
syntax = "proto3";

package p5.plugin.v0;

option go_package = "github.com/rfhold/p5/internal/plugins/proto";

service AuthPlugin {
  rpc Authenticate(AuthenticateRequest) returns (AuthenticateResponse);
}

message AuthenticateRequest {
  map<string, string> program_config = 1;
  map<string, string> stack_config = 2;
  string stack_name = 3;
  string program_name = 4;
}

message AuthenticateResponse {
  bool success = 1;
  map<string, string> env = 2;
  int32 ttl_seconds = 3;  // -1 = always call, 0 = never expires, >0 = TTL
  string error = 4;
}
```

## Implementation Tasks

### Phase 1: Core Infrastructure

1. **Add go-plugin dependency**
   - `go get github.com/hashicorp/go-plugin`

2. **Create proto definition**
   - `internal/plugins/proto/plugin.proto`
   - Generate Go code with `protoc`

3. **Implement plugin interface**
   - `internal/plugins/interface.go` - Define Go interface
   - `internal/plugins/grpc.go` - gRPC client/server implementations

4. **Create plugin shared library**
   - `pkg/plugin/` - Shared types and helpers for plugin authors
   - Include handshake config, serve function wrapper

### Phase 2: Update Plugin Manager

5. **Update config structures** (`manifest.go`)
   - Remove: `Source`, manifest loading, permissions
   - Add: `Cmd`, `Args`
   - Keep: `Config`, `Refresh` (refresh triggers still useful)

6. **Update manager** (`manager.go`)
   - Remove: `LoadPlugins`, WASM loading logic
   - Add: go-plugin `Client` management
   - Keep: Credential caching, TTL logic, context tracking

7. **Remove WASM runtime** (`runtime.go`)
   - Delete entirely (replaced by gRPC interface)

8. **Remove source handling** (`sources.go`)
   - Delete entirely (no more download/cache logic)

### Phase 3: Update Example Plugin

9. **Rewrite dev-env plugin**
   - Convert from TinyGo/WASM to native Go
   - Implement gRPC server
   - Remove manifest file

### Phase 4: Documentation

10. **Update docs**
    - Rewrite `docs/plugins.md` for new architecture
    - Remove `docs/extism.md` (no longer relevant)
    - Update README if needed

### Phase 5: Cleanup

11. **Remove Extism dependency**
    - Remove from `go.mod`
    - Clean up any remaining references

## File Changes Summary

### Delete
- `internal/plugins/runtime.go`
- `internal/plugins/sources.go`
- `plugins/dev-env/dev-env.wasm`
- `plugins/dev-env/dev-env.manifest.yaml`
- `docs/extism.md`

### Create
- `internal/plugins/proto/plugin.proto`
- `internal/plugins/proto/plugin.pb.go` (generated)
- `internal/plugins/proto/plugin_grpc.pb.go` (generated)
- `internal/plugins/interface.go`
- `internal/plugins/grpc.go`
- `pkg/plugin/plugin.go` (shared library for plugin authors)

### Modify
- `internal/plugins/manager.go` - Use go-plugin Client
- `internal/plugins/manifest.go` - Simplify config, remove manifest
- `plugins/dev-env/main.go` - Rewrite as gRPC plugin
- `docs/plugins.md` - Complete rewrite
- `go.mod` - Swap dependencies
- `p5.toml` - Update syntax

## Plugin Author Experience

### Before (TinyGo/WASM)

```go
package main

import (
    "encoding/json"
    "github.com/extism/go-pdk"
)

//go:wasmexport authenticate
func authenticate() int32 {
    inputBytes := pdk.Input()
    var input AuthInput
    json.Unmarshal(inputBytes, &input)
    
    // ... logic ...
    
    pdk.Output(outputBytes)
    return 0
}

func main() {}
```

Build: `tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared main.go`

### After (go-plugin)

```go
package main

import (
    "context"
    "github.com/rfhold/p5/pkg/plugin"
)

type MyPlugin struct{}

func (p *MyPlugin) Authenticate(ctx context.Context, req *plugin.AuthRequest) (*plugin.AuthResponse, error) {
    // ... logic (full Go stdlib available!) ...
    
    return &plugin.AuthResponse{
        Success: true,
        Env: map[string]string{
            "AWS_ACCESS_KEY_ID": "...",
        },
        TTLSeconds: 3600,
    }, nil
}

func main() {
    plugin.Serve(&MyPlugin{})
}
```

Build: `go build -o my-plugin .`

## Handshake Configuration

go-plugin uses a handshake to ensure host/plugin compatibility:

```go
var Handshake = plugin.HandshakeConfig{
    ProtocolVersion:  1,
    MagicCookieKey:   "P5_PLUGIN",
    MagicCookieValue: "v0",
}
```

## Security Considerations

With this migration, plugins have full system access. Document that:

1. Only install plugins from trusted sources
2. Review plugin source code if available
3. Plugins run with the same permissions as p5
4. Consider running p5 with minimal required permissions

## Testing Strategy

1. Unit tests for gRPC client/server
2. Integration test with dev-env plugin
3. Test plugin crash handling (go-plugin recovers gracefully)
4. Test credential caching still works
5. Test refresh triggers still work

## Rollout

1. Implement in feature branch
2. Update dev-env plugin first
3. Test with existing p5.toml configs (will need migration)
4. Provide migration guide for any external plugins
