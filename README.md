# P5

A TUI for managing Pulumi stacks.

![p5 demo](demo.gif)

## Install

```bash
go install github.com/rfhold/p5/cmd/p5@latest
```

## Usage

```bash
p5                    # Current directory
p5 -C /path/to/project
p5 -s dev             # Specific stack
p5 up                 # Start with up preview
p5 refresh            # Start with refresh preview
p5 destroy            # Start with destroy preview
```

## Keybindings

### Navigation
| Key | Action |
|-----|--------|
| `j`/`k` | Up/down |
| `g`/`G` | Top/bottom |
| `PgUp`/`PgDn` | Page scroll |

### Views
| Key | Action |
|-----|--------|
| `s` | Stack selector |
| `w` | Workspace selector |
| `h` | History view |
| `D` | Details panel |
| `?` | Help |

### Preview (lowercase)
| Key | Action |
|-----|--------|
| `u` | Preview up |
| `r` | Preview refresh |
| `d` | Preview destroy |

### Execute (uppercase)
| Key | Action |
|-----|--------|
| `U` | Execute up |
| `R` | Execute refresh |

### Flags
| Key | Action |
|-----|--------|
| `t` | Target |
| `p` | Replace |
| `x` | Exclude |
| `v` | Visual select |
| `c`/`C` | Clear flags |

### Actions
| Key | Action |
|-----|--------|
| `i` | Import (preview create ops) |
| `x` | Delete from state |
| `P` | Protect/unprotect |
| `o` | Open in external tool |
| `y`/`Y` | Copy JSON |
| `Esc` | Back/cancel |
| `q` | Quit |

## Plugins

Extend p5 with authentication, import helpers, and resource openers.

### Builtin
- **env**: Load environment variables
- **kubernetes**: Import suggestions via kubectl
- **k9s**: Open resources in k9s
- **grafana**: Open resources in browser
- **cloudflare**: Import suggestions (stub)

### Configuration

```toml
# p5.toml
[plugins.env.config]
path = ".env"
```

```yaml
# Pulumi.yaml
p5:
  plugins:
    kubernetes:
      import_helper: true
    k9s:
      resource_opener: true
```

See [docs/plugins/](docs/plugins/) for details.

## Documentation

- [Dependencies](docs/dependencies/) - Pulumi, Bubbletea integration
- [Development](docs/dev/) - Testing, releases
- [Plugins](docs/plugins/) - Plugin interface and builtins
- [Features](docs/features/) - Feature documentation

## Development

```bash
go build -o /dev/null ./cmd/p5 && ./scripts/dev.sh -C programs/simple
./scripts/view.sh  # View output
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for testing and contribution guidelines.

## License

[LICENSE](LICENSE)
