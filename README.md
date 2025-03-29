# p5
Pulumi was too long

A TUI application to help you manage your Pulumi stacks. It detects `Pulumi.yaml` files 2 levels deep from the current directory and allows you to perform operations on them.


MVP only allows for:

- Viewing programs
- Viewing stacks
- Creating stacks
- Deleting stacks
- Updating stacks
- Previewing stacks
- Listing Resources
- Importing Resources
- Auto Importing Resources
    - Implemented by resource type, Only basic k8s resources are supported at present
- Specify Provider for Importing Resources


see `handle_key` function in `main.py` for keybindings.

It will automatically switch your backend if `PULUMI_BACKEND` is set in your environment variables.

The secret provider for new stacks defaults to `passphrase` but can be changed with `PULUMI_SECRET_PROVIDER` in your environment variables.
