# Pulumi Automation SDK for Rust

A Rust implementation of the [Pulumi Automation API](https://www.pulumi.com/docs/iac/automation-api/), enabling programmatic infrastructure management without direct CLI invocation.

## Overview

This SDK provides a Rust interface for managing Pulumi stacks, configurations, and deployments. It wraps the Pulumi CLI and provides type-safe access to infrastructure operations with async event streaming.

## Installation

Add to your `Cargo.toml`:

```toml
[dependencies]
pulumi-automation = "0.8"
```

## Quick Start

```rust
use pulumi_automation::local::LocalWorkspace;
use pulumi_automation::workspace::Workspace;
use pulumi_automation::stack::{Stack, StackUpOptions, PulumiProcessListener};
use tokio::sync::mpsc;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Create a workspace pointing to your Pulumi project
    let workspace = LocalWorkspace::new("/path/to/pulumi/project".to_string());
    
    // Select or create a stack
    let stack = workspace.select_or_create_stack("dev", None)?;
    
    // Set up event listener
    let (event_tx, mut event_rx) = mpsc::channel(100);
    let listener = PulumiProcessListener {
        preview_tx: None,
        event_tx,
    };
    
    // Deploy the stack
    stack.up(StackUpOptions::default(), listener).await?;
    
    // Process events
    while let Some(event) = event_rx.recv().await {
        println!("Event: {:?}", event);
    }
    
    Ok(())
}
```

## Core Concepts

### Workspace

The `Workspace` trait defines the execution context for Pulumi operations. Currently, `LocalWorkspace` is the primary implementation, which executes operations using the local Pulumi CLI.

```rust
let workspace = LocalWorkspace::new("/path/to/project".to_string());
```

### Stack

A `Stack` represents an isolated, independently configurable instance of a Pulumi program. The `Stack` trait provides methods for lifecycle operations like `up`, `preview`, `refresh`, and `destroy`.

```rust
let stack = workspace.select_stack("production")?;
```

## API Reference

### Workspace Operations

#### Stack Management

| Method | Description |
|--------|-------------|
| `create_stack(name, options)` | Create a new stack |
| `select_stack(name)` | Select an existing stack |
| `select_or_create_stack(name, options)` | Select or create a stack |
| `remove_stack(name, options)` | Remove a stack |
| `list_stacks(options)` | List all stacks |

#### Configuration Management

| Method | Description |
|--------|-------------|
| `get_config(stack, key, path)` | Get a config value |
| `set_config(stack, key, path, value)` | Set a config value |
| `remove_config(stack, key, path)` | Remove a config value |
| `get_stack_config(stack)` | Get full stack settings |
| `set_stack_config(stack, config)` | Set stack settings |
| `stack_outputs(stack)` | Get stack outputs |

#### State Management

| Method | Description |
|--------|-------------|
| `export_stack(stack)` | Export stack deployment state |
| `import_stack(stack, deployment)` | Import stack deployment state |

#### Identity

| Method | Description |
|--------|-------------|
| `whoami()` | Get current user information |

### Stack Operations

All stack operations (except synchronous `preview`) are async and stream events via channels.

| Method | Description |
|--------|-------------|
| `preview(options)` | Preview changes (sync) |
| `preview_async(options, listener)` | Preview changes with event streaming |
| `up(options, listener)` | Deploy stack |
| `refresh(options, listener)` | Refresh stack state |
| `destroy(options, listener)` | Destroy stack resources |

### Event Types

The SDK provides comprehensive event streaming during operations:

| Event Type | Description |
|------------|-------------|
| `CancelEvent` | Operation cancelled |
| `Diagnostic` | Diagnostic message (debug, info, warning, error) |
| `PreludeEvent` | Operation starting |
| `SummaryEvent` | Operation completed |
| `ResourcePreEvent` | Resource about to be modified |
| `ResOutputsEvent` | Resource modification complete |
| `ResOpFailedEvent` | Resource operation failed |
| `StdoutEvent` | Standard output message |
| `PolicyEvent` | Policy violation |
| `StartDebuggingEvent` | Debugger attachment |
| `ProgressEvent` | Progress update |

## Feature Comparison with Official Node.js SDK

This table compares the Rust SDK features against the [official Pulumi Automation API for Node.js](https://www.pulumi.com/docs/reference/pkg/nodejs/pulumi/pulumi/automation/).

### Workspace Features

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| LocalWorkspace | Yes | Yes | |
| RemoteWorkspace | Yes | No | Pulumi Deployments not supported |
| Custom Workspace | Yes | Yes | Via `Workspace` trait |
| Environment Variables | Yes | No | Hardcoded empty |
| Custom Pulumi Home | Yes | No | |
| Secrets Provider Config | Yes | Partial | Via stack creation only |

### Stack Operations

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| up (deploy) | Yes | Yes | Async with events |
| preview | Yes | Yes | Sync and async versions |
| refresh | Yes | Yes | Async with events |
| destroy | Yes | Yes | Async with events |
| previewRefresh | Yes | No | |
| previewDestroy | Yes | No | |
| import (resources) | Yes | No | Resource import not supported |
| exportStack | Yes | Yes | |
| importStack | Yes | Yes | |
| cancel | Yes | No | |
| rename | Yes | No | |
| history | Yes | No | |
| outputs | Yes | Yes | Via `stack_outputs()` |

### Stack Management

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| createStack | Yes | Yes | |
| selectStack | Yes | Yes | |
| createOrSelectStack | Yes | Yes | |
| removeStack | Yes | Yes | With force/preserveConfig options |
| listStacks | Yes | Yes | |

### Configuration

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| getConfig | Yes | Yes | With path support |
| setConfig | Yes | Yes | With path and secret support |
| removeConfig | Yes | Yes | |
| getAllConfig | Yes | No | Use `get_stack_config` instead |
| setAllConfig | Yes | No | |
| removeAllConfig | Yes | No | |
| refreshConfig | Yes | No | |
| Path-based config | Yes | Yes | |
| Secrets | Yes | Yes | |

### Environment (ESC) Management

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| addEnvironments | Yes | No | |
| listEnvironments | Yes | No | |
| removeEnvironment | Yes | No | |

### Tag Management

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| getTag | Yes | No | |
| setTag | Yes | No | |
| removeTag | Yes | No | |
| listTags | Yes | No | |

### Plugin Management

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| installPlugin | Yes | No | |
| installPluginFromServer | Yes | No | |
| removePlugin | Yes | No | |
| listPlugins | Yes | No | |
| install (dependencies) | Yes | No | |

### Project & Stack Settings

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| projectSettings | Yes | No | Relies on existing Pulumi.yaml |
| saveProjectSettings | Yes | No | |
| stackSettings | Yes | Yes | Via `get_stack_config` |
| saveStackSettings | Yes | Yes | Via `set_stack_config` |

### Event Handling

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| onOutput callback | Yes | Yes | Via event channel |
| onError callback | Yes | Yes | Via event channel |
| onEvent callback | Yes | Yes | Via `PulumiProcessListener` |
| CancelEvent | Yes | Yes | |
| DiagnosticEvent | Yes | Yes | |
| PreludeEvent | Yes | Yes | |
| SummaryEvent | Yes | Yes | |
| ResourcePreEvent | Yes | Yes | |
| ResOutputsEvent | Yes | Yes | |
| ResOpFailedEvent | Yes | Yes | |
| PolicyEvent | Yes | Yes | |
| StdoutEvent | Yes | Yes | |
| StartDebuggingEvent | Yes | Yes | |
| ProgressEvent | No | Yes | Rust-specific |

### Operation Options

| Option | Node.js SDK | Rust SDK | Notes |
|--------|:-----------:|:--------:|-------|
| parallel | Yes | No | Parallelism control |
| message | Yes | No | Update message |
| expectNoChanges | Yes | Yes | |
| refresh | Yes | Yes | |
| diff | Yes | Yes | |
| replace | Yes | Yes | |
| target | Yes | Yes | |
| targetDependents | Yes | Yes | |
| exclude | Yes | Yes | |
| excludeDependents | Yes | Yes | |
| excludeProtected | Yes | Yes | Destroy only |
| policyPacks | Yes | No | |
| policyPackConfigs | Yes | No | |
| plan | Yes | No | Update plans |
| showSecrets | Yes | Yes | |
| continueOnError | Yes | Yes | |
| showReads | Yes | Yes | |
| showReplacementSteps | Yes | Yes | |
| AbortSignal (cancellation) | Yes | No | |

### Advanced Features

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| Inline programs | Yes | No | Functions as programs |
| Local/CLI programs | Yes | Yes | Primary mode |
| Remote operations | Yes | No | Pulumi Deployments |
| Parallel operations | Yes | No | |
| Update plans | Yes | No | |
| Policy packs | Yes | No | |
| Debugger attachment | Yes | Yes | Via events |

### Authentication

| Feature | Node.js SDK | Rust SDK | Notes |
|---------|:-----------:|:--------:|-------|
| whoAmI | Yes | Yes | With token info |

## Architecture

### Design Principles

1. **Trait-based extensibility**: `Workspace` and `Stack` traits allow for custom implementations
2. **Async-first**: Operations stream events via `tokio::sync::mpsc` channels
3. **Type safety**: Strong typing for all configuration values, events, and options
4. **Forward compatibility**: Unknown JSON fields are captured via `extra_values` fields

### Event Streaming

Unlike the callback-based Node.js SDK, this Rust SDK uses channels for event streaming:

```rust
let (event_tx, mut event_rx) = mpsc::channel(100);
let listener = PulumiProcessListener {
    preview_tx: None,
    event_tx,
};

// Start operation
let handle = tokio::spawn(async move {
    stack.up(options, listener).await
});

// Process events as they arrive
while let Some(event) = event_rx.recv().await {
    match event.event {
        EventType::ResourcePreEvent(details) => {
            println!("Modifying: {}", details.metadata.urn);
        }
        EventType::Diagnostic(details) => {
            println!("[{}] {}", details.severity, details.message);
        }
        _ => {}
    }
}

handle.await??;
```

## Dependencies

- `tokio` - Async runtime
- `serde` / `serde_json` / `serde_yaml` - Serialization
- `async-trait` - Async trait support
- `strum` - Enum utilities
- `tracing` - Instrumentation

## License

See [LICENSE](../LICENSE) for details.
