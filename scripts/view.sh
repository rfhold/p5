#!/bin/bash
# Capture and print the contents of tmux pane 0
# Useful for seeing what the app is currently displaying

tmux capture-pane -t :.0 -p
