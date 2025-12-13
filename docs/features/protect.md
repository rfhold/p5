# Resource Protection

![Protect Demo](../assets/protect.gif)

Protect resources from accidental destruction using Pulumi's built-in protection feature.

## Overview

Protected resources cannot be destroyed by Pulumi operations. This is useful for critical resources like databases, persistent volumes, or any infrastructure that should not be accidentally deleted.

## Keybinding

| Key | Action |
|-----|--------|
| `P` | Toggle protect/unprotect |

## Behavior

### Protecting a Resource

Pressing `P` on an unprotected resource **executes immediately** without confirmation. Protection is a safety action that prevents destruction.

### Unprotecting a Resource

Pressing `P` on a protected resource **shows a confirmation modal**. Since unprotecting makes the resource destroyable again, explicit confirmation is required.

## Display

Protected resources show a shield indicator `[🛡]` in the resource list.

## Restrictions

- Cannot protect the stack root resource
- Only available in stack view (not during preview/execute operations)

## CLI Equivalent

p5 uses the Pulumi CLI under the hood:

```bash
# Protect
pulumi state protect <urn>

# Unprotect
pulumi state unprotect <urn>
```

## Use Cases

- Protect production databases before running destroy
- Mark critical infrastructure as protected by default
- Temporarily unprotect a resource for replacement, then re-protect

## Implementation

- `cmd/p5/update_keys.go` - Key handler for `P`
- `cmd/p5/commands.go` - `executeProtect()` function
- `internal/pulumi/import.go` - `ProtectResource()` and `UnprotectResource()`
- `internal/ui/resourcerender.go` - Shield indicator display
