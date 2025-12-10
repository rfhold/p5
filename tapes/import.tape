# Import tape: import existing resource into Pulumi state
# Demonstrates: import.md

Output docs/assets/import.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set WaitTimeout 30s

# Setup: clean state with stack but no resources
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && cd /app"
Enter
Sleep 2s
Type "clear"
Enter
Sleep 500ms
Show

# Start p5 with up preview to see create operations
Type "p5 -C programs/simple up"
Enter

# Wait for preview done
Wait+Screen /done/
Sleep 2s

# Navigate to a resource with create operation
Down@200ms 1
Sleep 500ms

# Open import modal (I key - capital)
Type "I"

# Wait for import modal
Wait+Screen /Import Resource/
Sleep 2s

# Type import ID (fake ID for demo purposes)
Type "existing-resource-id-12345"
Sleep 1s

# Show entering the ID, then cancel (we don't want to actually import)
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
