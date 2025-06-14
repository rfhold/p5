# P5

Pulumi was too long

A TUI application to help you manage your Pulumi stacks.

## Installation

```bash
cargo install p5
```

## Demo

![Demo usage of p5](tapes/output/demo.gif)

## Features

- [x] Contexts
- [x] Command prompt
- [x] Select workspace
- [x] Select stack
- [x] Show stack information
    - [x] Show stack outputs
    - [x] Show stack settings
- [x] Show stack resources
    - [ ] Edit state json
        - [ ] Edit resource json
- [x] Preview stack changes
    - [x] Operation type colors
    - [ ] Detailed Diff
    - [ ] Special iconography
        - [ ] Protected resources
    - [ ] Resources Hierarchy
- [x] Update stack
- [x] Destroy stack
- [x] Refresh stack
- [ ] Cancel Operation
- [ ] Include and Exclude resources
- [ ] Import resources
- [ ] Remove resources
- [x] List workspaces
    - [x] Select workspace
    - [ ] Create workspace
- [x] List stacks
    - [x] Select stack
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
- [x] Navigation
- [ ] Context Command Palette
	- [ ] Keybinds
- [ ] Help
- [ ] Show stack history
- [ ] Event log
    - [ ] Show event log
    - [ ] Filter event log

## Motivation

Pulumi is a great tool, but the CLI is not very user friendly. I wanted to create a TUI application that would make it easier to manage Pulumi stacks and programs.
I also wanted to get a better grasp of async rust and TUI development, so this is a great opportunity to do both. With p5 I should be able to rapidly iterate over
IaC changes while also assisting in complicated sate manipulation.

## Debt

- [ ] pulumi-automation error handling and async cleanup
    - [ ] error stream for operations
- [ ] tracing
- [ ] otel
