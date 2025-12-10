# Pulumi CLI

p5 requires the Pulumi CLI for one operation: state deletion.

## Requirement

- Pulumi CLI must be installed and available in PATH
- Used by `ResourceImporter.StateDelete()` in `internal/pulumi/default_importer.go`

## Usage

```bash
pulumi state delete <urn> --stack <stack> --yes
```

This command removes a resource from Pulumi state without destroying the actual cloud resource. The `--yes` flag bypasses confirmation prompts.

## Why CLI Instead of Automation API

The Pulumi Automation API does not expose a state delete operation. The CLI is invoked directly for this specific functionality.

## Related

- [Pulumi Automation API](pulumi-automation-api.md) - Used for all other operations
