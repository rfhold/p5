# History tape: view stack history and history details
# Demonstrates: history.md

Output docs/assets/history.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# Setup: create resources with history
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && pulumi up -y && sleep 2 && pulumi up -y && cd /app"
Enter
Sleep 15s
Type "clear"
Enter
Sleep 500ms
Show

# Start p5
Type "p5 -C programs/simple"
Enter
Sleep 500ms

# Wait for preview
Wait+Screen /resources/
Sleep 1s

# Open history view
Type "h"
Sleep 1s

# Wait for history to load
Wait+Screen /history/
Sleep 1s

# Navigate history entries
Down@200ms 1
Sleep 500ms

# View history details
Type "D"
Sleep 2s

# Scroll through details
Down@200ms 3
Sleep 1s

# Go back to history list
Escape
Sleep 500ms

# Go back to resources
Escape
Sleep 500ms

# Quit
Type "q"
Sleep 1s
