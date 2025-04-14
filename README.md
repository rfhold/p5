# p5

Pulumi was too long

A TUI application to help you manage your Pulumi stacks. It detects `Pulumi.yaml` files 2 levels deep from the current directory and allows you to perform operations on them.

## Features

- [x] List Programs
- [x] Select Program
- [x] List Stacks
- [x] Select Stack
- [ ] Show Stack Config
- [x] Show Stack Outputs
- [ ] Show Stack History
- [x] Preview Stack
- [ ] Update Stack
- [ ] Destroy Stack
- [ ] Refresh Stack
- [ ] Delete Stack
- [ ] Export Stack
- [ ] Import Stack
- [ ] New Stack
- [ ] Rename Stack
- [ ] Change Stack Secrets Provider
- [ ] Move Stack
- [ ] Stack Graph
- [ ] List Resources
- [ ] Show Resource State
- [ ] Rename Resource
- [ ] Import Resource
- [ ] Move Resource
- [ ] Select Resource(s)
- [ ] Preview Resource(s)
- [ ] Delete Resource(s)
- [ ] Refresh Resource(s)
- [ ] Destroy Resource(s)
- [ ] Update Resource(s)
- [ ] Resource Graph
- [ ] Automatic Help Menu

### Brainstorm Features

- [ ] Docker Pulumi Exection Mode
- [ ] Parralel Program/Stack Operations
- [ ] Auto Import
- [ ] Workflows
- [ ] External Links
- [ ] Backend configuration per Program and/or Stack
- [ ] Authentication Command hook configurable 

## State

WIP, usable but needs polishing. The State machine that handles events, actions, and rendering needs to be refactored. 

## Installation

```bash
cargo install p5
```

## Motivation

Pulumi is a great tool, but the CLI is not very user friendly. I wanted to create a TUI application that would make it easier to manage Pulumi stacks and programs.
I also wanted to get a better grasp of async rust and TUI development, so this is a great opportunity to do both. With p5 I should be able to rapidly iterate over
IaC changes while also assisting in complicated sate manipulation.
