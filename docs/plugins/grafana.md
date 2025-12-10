# Grafana Plugin

Builtin plugin for opening Grafana resources in browser.

## Capabilities

- **Resource Opener**: Opens Grafana resources in default browser

## Configuration

Requires `grafana:url` in provider inputs, stack config, or program config.

```yaml
# Pulumi.yaml
p5:
  plugins:
    grafana:
      resource_opener: true
      config:
        url: https://grafana.example.com
```

## Supported Resources

| Resource Type | URL Pattern |
|--------------|-------------|
| `grafana:onCall/escalationChain:EscalationChain` | `/a/grafana-irm-app/escalations/{id}` |
| `grafana:onCall/integration:Integration` | `/a/grafana-irm-app/integrations/{id}` |
| `grafana:onCall/schedule:Schedule` | `/a/grafana-irm-app/schedules/{id}` |
| `grafana:oss/team:Team` | `/org/teams/edit/{teamUid}` |
| `grafana:oss/dashboard:Dashboard` | Uses `url` output directly |
| `grafana:alerting/ruleGroup:RuleGroup` | `/alerting/grafana/namespaces/{folder}/groups/{name}/view` |
| `grafana:alerting/contactPoint:ContactPoint` | `/alerting/notifications` |
| `grafana:cloud/accessPolicy:AccessPolicy` | `https://grafana.com/orgs/{slug}/access-policies` |

## Usage

1. Enable resource opener in config
2. Navigate to a Grafana resource in p5
3. Press `o` to open in browser

## Implementation

Located in `internal/plugins/builtins/grafana.go`.
