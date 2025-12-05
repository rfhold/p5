# P5

Pulumi was too long.

A TUI application to help you manage your Pulumi stacks.

> **Note**: This project (including this README) was largely generated with the assistance of LLMs.

![p5 demo](demo.gif)

## Installation

```bash
go install github.com/rfhold/p5/cmd/p5@latest
```

## Features

- **Stack Management**: View and switch between Pulumi stacks
- **Resource Browser**: Explore resources in your stack with a tree view
- **Preview Operations**: Run `up`, `refresh`, and `destroy` previews
- **Execute Operations**: Apply changes directly from the TUI
- **Visual Selection**: Select multiple resources for targeted operations
- **Resource Import**: Import existing resources into Pulumi state
- **State Management**: Delete resources from state
- **History View**: Browse stack update history
- **Plugin System**: Extensible authentication via plugins

## Usage

```bash
# Start in current directory
p5

# Start in a specific directory
p5 -C /path/to/pulumi/project

# Start with a specific stack
p5 -s dev

# Start with a preview operation
p5 up       # Start with up preview
p5 refresh  # Start with refresh preview
p5 destroy  # Start with destroy preview
```

### Command Line Options

| Flag | Description |
|------|-------------|
| `-C`, `--cwd` | Run as if p5 was started in the specified directory |
| `-s`, `--stack` | Select the Pulumi stack to use |

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `g` / `Home` | Go to top |
| `G` / `End` | Go to bottom |
| `PgUp` / `Ctrl+b` | Page up |
| `PgDn` / `Ctrl+f` | Page down |

### Views

| Key | Action |
|-----|--------|
| `s` | Open stack selector |
| `w` | Open workspace selector |
| `h` | View stack history |
| `D` | Toggle details panel |
| `Esc` | Go back / Cancel |

### Preview Operations (lowercase)

| Key | Action |
|-----|--------|
| `u` | Preview up |
| `r` | Preview refresh |
| `d` | Preview destroy |

### Execute Operations (Ctrl+key)

| Key | Action |
|-----|--------|
| `Ctrl+u` | Execute up |
| `Ctrl+r` | Execute refresh |
| `Ctrl+d` | Execute destroy |

### Resource Flags

| Key | Action |
|-----|--------|
| `T` | Toggle target flag (--target) |
| `R` | Toggle replace flag (--replace) |
| `E` | Toggle exclude flag |
| `c` | Clear flags on current resource |
| `C` | Clear all flags |

### Visual Selection

| Key | Action |
|-----|--------|
| `v` | Enter visual selection mode |
| `Esc` | Exit visual mode |

Use visual mode to select multiple resources, then apply flags or operations to all selected.

### Resource Actions

| Key | Action |
|-----|--------|
| `I` | Import resource (in preview, on create operations) |
| `x` | Delete from state (in stack view) |
| `y` | Copy resource JSON to clipboard |
| `Y` | Copy all resources JSON to clipboard |

### General

| Key | Action |
|-----|--------|
| `?` | Show help |
| `q` / `Ctrl+c` | Quit |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         p5 TUI                              │
├─────────────────────────────────────────────────────────────┤
│  cmd/p5/           - Bubble Tea application                 │
│  ├── model.go      - State management and types             │
│  ├── view.go       - Rendering logic                        │
│  ├── commands.go   - Async operations (Bubble Tea commands) │
│  ├── messages.go   - Message types for update loop          │
│  └── update_*.go   - Update handlers by domain              │
├─────────────────────────────────────────────────────────────┤
│  internal/ui/      - Reusable UI components                 │
│  ├── resourcelist  - Resource browser with tree view        │
│  ├── details       - Resource detail panel                  │
│  ├── modals        - Import, confirm, error dialogs         │
│  └── selectors     - Stack and workspace selectors          │
├─────────────────────────────────────────────────────────────┤
│  internal/pulumi/  - Pulumi Automation API integration      │
│  ├── operations    - Preview and execute operations         │
│  ├── resources     - Stack resource queries                 │
│  └── history       - Stack update history                   │
├─────────────────────────────────────────────────────────────┤
│  internal/plugins/ - Plugin system for authentication       │
│  ├── builtins      - In-process plugins (env, kubernetes)   │
│  └── grpc          - External plugin support via go-plugin  │
└─────────────────────────────────────────────────────────────┘
```

## Plugin System

p5 supports plugins for authentication with configurable program and stack settings. Plugins can access both program-level configuration (from `Pulumi.yaml`) and stack-specific configuration (from `Pulumi.{stack}.yaml`), allowing for flexible authentication across different environments.

### Builtin Plugins

- **env**: Load environment variables from files, static config, or commands
- **kubernetes**: Kubernetes context management

### Example Configuration

```toml
# p5.toml (global defaults)
[plugins.env.config]
path = ".env"
```

```yaml
# Pulumi.yaml (program-level)
p5:
  plugins:
    env:
      config:
        sources:
          - type: "file"
            path: ".env.defaults"
```

```yaml
# Pulumi.dev.yaml (stack-level)
config:
  aws:region: us-west-2
  p5:plugins:
    env:
      config:
        sources: '[{"type":"file","path":".env.dev"}]'
```

See [docs/plugins.md](docs/plugins.md) for detailed plugin documentation including:
- Complete configuration options and precedence
- External plugin development guide
- Program and stack configurability examples
- Authentication lifecycle and caching

## Motivation

Pulumi is a great tool, but the CLI is not very user friendly. I wanted to create a TUI application that would make it easier to manage Pulumi stacks and programs. With p5 I should be able to rapidly iterate over IaC changes while also assisting in complicated state manipulation.

## Development

### Prerequisites

- Go 1.21+
- A Pulumi project for testing

### Running Locally

```bash
# Build and run
go build -o /dev/null ./cmd/p5 && ./scripts/dev.sh -C programs/simple

# View output in tmux pane
./scripts/view.sh
```

### Project Structure

```
.
├── cmd/p5/          # Main application
├── internal/
│   ├── plugins/     # Plugin system
│   ├── pulumi/      # Pulumi API integration
│   └── ui/          # UI components
├── pkg/plugin/      # Public plugin interface
├── docs/            # Documentation
└── test/            # Test Pulumi projects
```

## Recording Demo

The demo GIF is recorded using [VHS](https://github.com/charmbracelet/vhs). To regenerate it:

```bash
# Build the VHS Docker image
docker build -f Dockerfile.vhs -t p5-vhs .

# Run VHS to generate demo.gif
docker run --rm -v "$(pwd)":/app p5-vhs
```

## License

See [LICENSE](LICENSE) for details.
