# Stack Initialization

Create new stacks or select existing ones.

![Stack Initialization](../assets/stack-init.gif)

## Stack Selection

Press `s` to open stack selector.

Lists:
- Backend stacks from Pulumi
- File-based stacks from `Pulumi.*.yaml` files

## Stack Creation

If no stacks exist, stack init modal opens automatically.

Modal collects:
- **Stack name**: Required, e.g., `dev`, `prod`
- **Secrets provider**: Optional, e.g., `awskms://...`, `passphrase`
- **Passphrase**: Required if secrets provider is `passphrase`

## Flow

1. Check for existing stacks
2. If none: show init modal
3. Collect stack name and secrets provider
4. Call `StackInitializer.InitStack()`
5. Select new stack
6. Load resources

## Secrets Providers

Supported providers:
- `default` - Uses Pulumi Service encryption
- `passphrase` - Local passphrase-based encryption
- `awskms://keyId` - AWS KMS
- `azurekeyvault://...` - Azure Key Vault
- `gcpkms://...` - Google Cloud KMS
- `hashivault://...` - HashiCorp Vault

## Stack Switching

When selecting a different stack:
1. Plugin credentials are invalidated (if configured)
2. Re-authentication runs
3. Resources reload for new stack

## Implementation

- `cmd/p5/update_init.go` - Initialization state machine
- `cmd/p5/update_selection.go` - Stack selection handlers
- `internal/ui/stackinitmodal.go` - Init modal component
- `internal/ui/stackselector.go` - Stack selector component
