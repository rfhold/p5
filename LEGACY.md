# P5 Legacy TUI Documentation

This document describes the features and architecture of the original P5 TUI application before the overhaul.

## Overview

P5 is a Terminal User Interface for managing Pulumi infrastructure-as-code stacks. The name "P5" comes from "Pulumi" being "too long."

## Navigation & Context System

### Context Stack

The application uses a hierarchical context stack for navigation:

```
WorkspaceList
  └── StackList
        └── Stack
              ├── Config
              ├── Outputs
              ├── Resources
              └── Operation
                    ├── Summary
                    └── Events
```

The context stack enables backward navigation, exit behavior, and conditional rendering.

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `Down Arrow` | Navigate down in lists |
| `k` / `Up Arrow` | Navigate up in lists |
| `h` / `Left Arrow` | Go back / Pop context |
| `l` / `Right Arrow` | Select / Enter context |
| `Esc` | Go back / Cancel |
| `Ctrl+C` | Exit application |
| `:` | Open command prompt |

## Views & Screens

### Main Layout

The interface is split into:
- **Left Sidebar (25%)**: Workspace List (top) and Stack List (bottom)
- **Main Area (75%)**: Stack details and operations

### Workspace List

- Auto-discovers Pulumi workspaces via `**/Pulumi.yaml` glob pattern
- Shows loading state while discovering
- Highlights currently selected workspace

### Stack List

- Lists all stacks within the selected workspace
- Shows "No Workspace Selected" when no workspace is active
- Shows loading state while fetching stacks

### Stack Detail Views

Four sub-views accessible via commands:

1. **Config** (`:config`) - Stack configuration in JSON format
2. **Outputs** (`:outputs`) - Stack outputs in JSON format
3. **Resources** (`:resources`) - Scrollable list of deployed resources
4. **Operation** - Active operation view with Summary and Events panels

### Operation View

When running an operation:
- **Summary Panel**: Preview of changes with operation types
- **Events Panel**: Real-time progress of resource operations
- **Details Panel**: Property-level diff of selected resource (via `:details`)

### Overlays

- **Command Prompt**: Centered popup for text input
- **Toast Notifications**: Bottom-right auto-dismissing errors (3 seconds)

## Pulumi Operations

| Command | Description |
|---------|-------------|
| `:preview` | Show what changes would be made without executing |
| `:update` | Execute a stack update (deploy changes) |
| `:refresh` | Synchronize state with actual cloud resources |
| `:destroy` | Tear down all resources in the stack |

### Operation Status Display

Operations show:
- Operation type (Update/Preview/Refresh/Destroy)
- Current status (Loading Summary, In Progress, Complete)
- Per-resource status (InProgress, Completed, Failed)
- Duration timing for each resource

## Commands

Available via `:` prompt:

| Command | Description |
|---------|-------------|
| `workspaces` | Navigate to workspace list |
| `workspace <path>` | Select a specific workspace by path |
| `stack <name>` | Select a specific stack by name |
| `outputs` | View stack outputs |
| `config` | View stack configuration |
| `resources` | View stack resources |
| `preview` | Run a preview operation |
| `update` | Run an update operation |
| `refresh` | Run a refresh operation |
| `destroy` | Run a destroy operation |
| `details` | Show detailed view for selected item |

## Visual Feedback & Theming

### Operation Type Colors

| Operation | Color |
|-----------|-------|
| Create | Green |
| Update | Yellow |
| Delete | Red |
| Refresh/Read | Blue |
| Replace | Magenta |
| Import | Light Green |
| Discard | Light Red |
| Same (no change) | Dark Gray |

### Resource States

- **Excluded**: Red with strikethrough
- **Targeted**: Green and bold
- **Replace**: Yellow and italic

### Operation Progress States

- **In Progress**: Default gray
- **Completed**: Dark gray
- **Failed**: Light red

### Details Panel Diff

- Additions: Green with `+` prefix
- Deletions: Red with `-` prefix
- Updates: Yellow with `~` prefix

## Loading States

The `Loadable` enum represents async data:
- **NotLoaded**: Initial state, no data fetched
- **Loading**: Data being fetched, shows loading indicator
- **Loaded**: Data available for display

## Architecture Patterns

### Events

Terminal events (key presses) that capture user input. Events can mutate state, trigger Actions, or do nothing.

### Actions

Synchronous operations performed on state. Triggered by Events or Tasks. Minimal computation, works with current state only.

### Tasks

Asynchronous operations triggered by Actions. Run outside state/event/rendering locks. Dispatch Actions when state updates are needed.

### Layout

Widgets for navigational conditional rendering, popups, and guards.

### Context

Represents user focus. Stored in `context_stack` for backward navigation, exit behavior, and conditional rendering.

## Unimplemented Features

These features were planned but not implemented:
- Cancel operation
- Include/Exclude resources from operations
- Import/Remove resources
- Create/Rename/Delete/Copy stacks
- Create workspaces
- Self-host configuration (backend URL, authentication)
- Edit Pulumi config
- Context-aware command palette with keybinds
- Help system
- Stack history
- Event log filtering
