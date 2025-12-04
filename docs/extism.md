# Extism Plugin Framework

Extism is a universal plug-in system that makes software programmable through WebAssembly. It provides cross-language support, cross-platform execution, security by default, and easy integration.

## Architecture Overview

| Component | Description |
|-----------|-------------|
| **Host** | Your application that loads and runs plugins |
| **Host SDK** | Library used by the host to manage and execute plugins (github.com/extism/go-sdk) |
| **Guest/Plugin** | WebAssembly code module (`.wasm` file) that extends functionality |
| **PDK** | Plugin Development Kit - library compiled into plugins to interact with the host |

```
Host Application (Go)
        ↓
   Host SDK (extism/go-sdk)
        ↓
   Wasm Runtime (wazero - pure Go, no CGO)
        ↓
   Plugin (.wasm) with PDK
```

## Go SDK Usage

### Installation

```bash
go get github.com/extism/go-sdk
```

### Creating and Loading Plugins

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/extism/go-sdk"
)

func main() {
    manifest := extism.Manifest{
        Wasm: []extism.Wasm{
            extism.WasmFile{Path: "./plugin.wasm"},
        },
    }

    ctx := context.Background()
    config := extism.PluginConfig{
        EnableWasi: true,
    }
    
    plugin, err := extism.NewPlugin(ctx, manifest, config, []extism.HostFunction{})
    if err != nil {
        fmt.Printf("Failed to initialize plugin: %v\n", err)
        os.Exit(1)
    }
    defer plugin.Close(ctx)

    // Call a plugin function
    exit, out, err := plugin.Call("greet", []byte("World"))
    if err != nil {
        fmt.Println(err)
        os.Exit(int(exit))
    }
    fmt.Println(string(out))
}
```

### Wasm Loading Options

```go
// From file
extism.WasmFile{
    Path: "./plugin.wasm",
    Hash: "sha256:abc123...",  // Optional integrity check
}

// From URL
extism.WasmUrl{
    Url:     "https://example.com/plugin.wasm",
    Headers: map[string]string{"Authorization": "Bearer token"},
    Hash:    "sha256:abc123...",
}

// From bytes
extism.WasmData{
    Data: wasmBytes,
}
```

## Manifest System

The manifest configures plugin security and resource limits:

```go
manifest := extism.Manifest{
    Wasm: []extism.Wasm{
        extism.WasmFile{Path: "plugin.wasm"},
    },
    
    // HTTP permissions (empty = none allowed, nil = all allowed)
    AllowedHosts: []string{
        "api.example.com",
        "*.github.com",  // Wildcard support
    },
    
    // Filesystem mappings (host path → plugin path)
    AllowedPaths: map[string]string{
        "/host/data": "/mnt",
    },
    
    // Static configuration for plugins
    Config: map[string]string{
        "api_key": "secret",
    },
    
    // Resource limits
    Memory: &extism.ManifestMemory{
        MaxPages:             16,        // 16 * 64KB = 1MB
        MaxHttpResponseBytes: 1048576,   // 1MB
        MaxVarBytes:          4096,      // 4KB
    },
    
    // Execution timeout
    Timeout: 5000,  // milliseconds
}
```

## Host Functions

Host functions allow plugins to call back into the host application.

### Defining Host Functions

```go
// Key-value store example
kvStore := make(map[string][]byte)

kvRead := extism.NewHostFunctionWithStack(
    "kv_read",
    func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
        key, _ := p.ReadString(stack[0])
        value := kvStore[key]
        stack[0], _ = p.WriteBytes(value)
    },
    []extism.ValueType{extism.ValueTypePTR},  // Input types
    []extism.ValueType{extism.ValueTypePTR},  // Output types
)

kvWrite := extism.NewHostFunctionWithStack(
    "kv_write",
    func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
        key, _ := p.ReadString(stack[0])
        value, _ := p.ReadBytes(stack[1])
        kvStore[key] = value
    },
    []extism.ValueType{extism.ValueTypePTR, extism.ValueTypePTR},
    []extism.ValueType{},
)

// Register when creating plugin
plugin, _ := extism.NewPlugin(ctx, manifest, config, 
    []extism.HostFunction{kvRead, kvWrite})
```

### CurrentPlugin Methods

| Method | Description |
|--------|-------------|
| `ReadString(offset)` | Read string from plugin memory |
| `ReadBytes(offset)` | Read bytes from plugin memory |
| `WriteString(s)` | Write string, returns offset |
| `WriteBytes(b)` | Write bytes, returns offset |
| `Alloc(n)` | Allocate n bytes, returns offset |
| `Free(offset)` | Free memory block |

### Value Types

```go
extism.ValueTypeI32   // 32-bit integer
extism.ValueTypeI64   // 64-bit integer
extism.ValueTypeF32   // 32-bit float
extism.ValueTypeF64   // 64-bit float
extism.ValueTypePTR   // Pointer (alias for I64)
```

## Plugin Development (TinyGo)

### Installation

Install [TinyGo](https://tinygo.org/getting-started/install/) and the PDK:

```bash
go get github.com/extism/go-pdk
```

### Writing a Plugin

```go
package main

import (
    "github.com/extism/go-pdk"
)

//go:wasmexport greet
func greet() int32 {
    input := pdk.Input()
    greeting := "Hello, " + string(input) + "!"
    pdk.OutputString(greeting)
    return 0  // Success
}

func main() {}
```

### Building

```bash
tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared main.go
```

### Plugin Features

**Configuration access:**
```go
//go:wasmexport greet
func greet() int32 {
    user, ok := pdk.GetConfig("user")
    if !ok {
        pdk.SetErrorString("'user' config required")
        return 1
    }
    pdk.OutputString("Hello, " + user + "!")
    return 0
}
```

**Variables (persistent state):**
```go
//go:wasmexport count
func count() int32 {
    n := pdk.GetVarInt("count")
    n++
    pdk.SetVarInt("count", n)
    pdk.OutputString(strconv.Itoa(n))
    return 0
}
```

**HTTP requests:**
```go
//go:wasmexport fetch
func fetch() int32 {
    req := pdk.NewHTTPRequest(pdk.MethodGet, "https://api.example.com/data")
    req.SetHeader("Authorization", "Bearer token")
    res := req.Send()
    pdk.OutputMemory(res.Memory())
    return 0
}
```

**Calling host functions:**
```go
//go:wasmimport extism:host/user kv_read
func kvRead(uint64) uint64

//go:wasmexport process
func process() int32 {
    mem := pdk.AllocateString("my-key")
    defer mem.Free()
    
    ptr := kvRead(mem.Offset())
    result := pdk.FindMemory(ptr)
    
    pdk.Output(result.ReadBytes())
    return 0
}
```

**Error handling:**
```go
//go:wasmexport process
func process() int32 {
    result, err := doSomething()
    if err != nil {
        pdk.SetError(err)
        return 1
    }
    pdk.Output(result)
    return 0
}
```

**JSON handling:**
```go
type Request struct {
    Name string `json:"name"`
}

type Response struct {
    Message string `json:"message"`
}

//go:wasmexport handle
func handle() int32 {
    var req Request
    if err := pdk.InputJSON(&req); err != nil {
        pdk.SetError(err)
        return 1
    }
    
    resp := Response{Message: "Hello, " + req.Name}
    if _, err := pdk.OutputJSON(resp); err != nil {
        pdk.SetError(err)
        return 1
    }
    return 0
}
```

## Security Model

Extism follows a **deny-by-default** security model:

| Capability | Default | Enable Via |
|------------|---------|------------|
| HTTP Requests | Denied | `AllowedHosts` in manifest |
| Filesystem | Denied | `AllowedPaths` + `EnableWasi` |
| Host Functions | None | Explicit registration |
| WASI | Disabled | `EnableWasi: true` |

**Best Practices:**
- Start with minimal permissions
- Use hash verification for remote plugins
- Set memory limits and timeouts
- Audit all host functions

## Advanced Usage

### Compilation Caching

```go
import "github.com/tetratelabs/wazero"

cache := wazero.NewCompilationCache()
defer cache.Close(ctx)

config := extism.PluginConfig{
    RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(cache),
}
```

### Concurrent Execution

```go
// Compile once
compiled, _ := extism.NewCompiledPlugin(ctx, manifest, config, hostFuncs)
defer compiled.Close(ctx)

// Create instances for concurrent use
for i := 0; i < 10; i++ {
    go func() {
        instance, _ := compiled.Instance(ctx, extism.PluginInstanceConfig{})
        defer instance.Close(ctx)
        
        _, out, _ := instance.Call("process", data)
        fmt.Println(string(out))
    }()
}
```

### Error Handling

```go
exit, out, err := plugin.Call("function", input)
if err != nil {
    // Execution error (timeout, trap, etc.)
    log.Printf("Execution error: %v", err)
    return
}

if exit != 0 {
    // Plugin returned error
    errMsg := plugin.GetError()
    log.Printf("Plugin error (code %d): %s", exit, errMsg)
    return
}

// Success
result := string(out)
```

### Logging

```go
// Host side
plugin.SetLogger(func(level extism.LogLevel, message string) {
    log.Printf("[%s] %s", level.String(), message)
})

// Plugin side
pdk.Log(pdk.LogInfo, "An info message")
pdk.Log(pdk.LogError, "Error occurred")
```

## Plugin Languages

| Language | PDK Repository | Build Command |
|----------|----------------|---------------|
| Go (TinyGo) | github.com/extism/go-pdk | `tinygo build -target wasip1 -buildmode=c-shared` |
| Rust | github.com/extism/rust-pdk | `cargo build --target wasm32-unknown-unknown` |
| JavaScript | github.com/extism/js-pdk | Via Extism CLI |
| AssemblyScript | github.com/extism/assemblyscript-pdk | `asc` |
| C | github.com/extism/c-pdk | `clang --target=wasm32-wasi` |

## References

- [Extism Documentation](https://extism.org/docs/)
- [Go SDK](https://github.com/extism/go-sdk)
- [Go PDK](https://github.com/extism/go-pdk)
- [API Reference](https://pkg.go.dev/github.com/extism/go-sdk)
