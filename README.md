# p5

Pulumi was too long

A TUI application to help you manage your Pulumi stacks.

## Features

- [ ] Contexts
- [ ] Command prompt
- [ ] Help
- [ ] Select program
- [ ] Select stack
- [ ] Show stack information
    - [ ] Show stack outputs
    - [ ] Show stack settings
- [ ] Show stack resources
    - [ ] Show state json
    - [ ] Edit state json
        - [ ] Edit resource json
- [ ] Preview stack changes
    - [ ] Operation type colors
    - [ ] Detailed Diff
    - [ ] Special iconography
        - [ ] Protected resources
    - [ ] Component resources
- [ ] Update stack
- [ ] Destroy stack
- [ ] Refresh stack
- [ ] Include and Exclude resources
- [ ] Import resources
- [ ] Remove resources
- [ ] List programs
    - [ ] Select program
    - [ ] Create program
- [ ] List stacks
    - [ ] Select stack
    - [ ] Create stack
    - [ ] Rename stack
    - [ ] Delete stack
    - [ ] Copy stack
- [ ] Self host config
    - [ ] Backend Url
    - [ ] Env
        - [ ] Authentication hook
        - [ ] Static
        - [ ] Secret Manager
- [ ] Edit Pulumi config
    - [ ] Edit stack config
    - [ ] Edit program config
- [ ] Show stack history


## Installation

```bash
cargo install p5
```

## Motivation

Pulumi is a great tool, but the CLI is not very user friendly. I wanted to create a TUI application that would make it easier to manage Pulumi stacks and programs.
I also wanted to get a better grasp of async rust and TUI development, so this is a great opportunity to do both. With p5 I should be able to rapidly iterate over
IaC changes while also assisting in complicated sate manipulation.
