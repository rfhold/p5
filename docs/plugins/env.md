# Env Plugin

Builtin plugin for loading environment variables. Primary use: authentication.

## Capabilities

- **Authentication**: Provides environment variables from multiple sources

## Configuration

```yaml
# Pulumi.yaml
p5:
  plugins:
    env:
      config:
        sources:
          - type: file
            path: ~/.secrets/my-env
          - type: static
            vars:
              FOO: bar
          - type: exec
            command: ["/path/to/script"]
```

## Source Types

### file

Reads `.env` format files (KEY=VALUE):

```yaml
config:
  type: file
  path: ~/.secrets/my-env
```

Supports `~` expansion for home directory.

### static

Inline variables from config:

```yaml
config:
  type: static
  vars:
    AWS_REGION: us-west-2
    DEBUG: "true"
```

### exec

Runs a command and parses stdout as .env format:

```yaml
config:
  type: exec
  command: ["/path/to/script", "--arg"]
```

## Multiple Sources

Sources are processed in order. Later sources override earlier ones:

```yaml
config:
  sources:
    - type: file
      path: .env.defaults
    - type: file
      path: .env.local
    - type: static
      vars:
        OVERRIDE: value
```

## Stack-Specific Config

```yaml
# Pulumi.dev.yaml
config:
  p5:plugins:
    env:
      config:
        sources: '[{"type":"file","path":".env.dev"}]'
```

## TTL

Environment variables never expire (TTL=0) by default. They are refreshed on workspace or stack change unless `refresh.on_config_change` is set.

## Implementation

Located in `internal/plugins/builtins/env.go`.
