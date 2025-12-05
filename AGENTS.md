# Agent Instructions

## Running the App

This is a Go TUI app using Bubble Tea and Pulumi Automation API. To check compilation and run it:

```bash
go build -o /dev/null ./cmd/p5 && ./scripts/dev.sh -C programs/simple # builds and runs the app
```

## Development Scripts

Launch the app in tmux pane 0:

```bash
./scripts/dev.sh
```

View the current output of the app in tmux pane 0:

```bash
./scripts/view.sh
```

## Project Structure

- `cmd/p5/main.go` - Bubble Tea TUI app with Pulumi Automation API integration
- `scripts/dev.sh` - Launch app in tmux pane 0
- `scripts/view.sh` - Capture tmux pane 0 output
