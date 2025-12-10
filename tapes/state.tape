# State tape: delete resource from state
# Demonstrates: state.md

Output docs/assets/state.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set WaitTimeout 30s

# Setup: create resources first
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && pulumi up -y && cd /app"
Enter
Sleep 10s
Type "clear"
Enter
Sleep 500ms
Show

# Start p5 in stack view (no operation mode)
Type "p5 -C programs/simple"
Enter
Sleep 500ms

# Wait for resources to load
Wait+Screen /resources/
Sleep 2s

# Navigate to a resource
Down@200ms 2
Sleep 1s

# Press x to delete from state
Type "x"

# Wait for confirmation modal
Wait+Screen /Delete from State/
Sleep 2s

# Show the confirmation, then cancel (don't actually delete)
Type "n"
Sleep 1s

# Navigate to another resource and show the flow again
Down@200ms 1
Sleep 500ms
Type "x"
Wait+Screen /Delete from State/
Sleep 2s

# This time confirm the delete
Type "y"

# Wait for deletion to complete (toast or resources update)
Sleep 3s

# Quit
Type "q"
Sleep 1s
