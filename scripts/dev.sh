#!/bin/bash
# Launch the app in tmux pane 0
# Safely checks if pane 0 is a shell or already running the app before proceeding
#
# Usage: ./scripts/dev.sh [args...]
#   All arguments are passed directly to p5

pane_cmd=$(tmux display-message -t :.0 -p '#{pane_current_command}')

case "$pane_cmd" in
  fish|bash|zsh|go|p5)
    tmux send-keys -t :.0 C-c
    sleep 0.2
    tmux send-keys -t :.0 "go run ./cmd/p5 $*" Enter
    ;;
  *)
    echo "Pane 0 is running '$pane_cmd', not a shell or p5. Skipping."
    ;;
esac
