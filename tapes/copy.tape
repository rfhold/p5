# Copy tape: copy resource data to clipboard
# Demonstrates: copy.md

Output docs/assets/copy.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# Setup: create resources
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && pulumi up -y && cd /app"
Enter
Sleep 10s
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

# Navigate to a resource
Down@200ms 2
Sleep 500ms

# Copy single resource (y)
Type "y"
Sleep 2s

# Navigate to another
Down@200ms 1
Sleep 500ms

# Copy all resources (Y)
Type "Y"
Sleep 2s

# Quit
Type "q"
Sleep 1s
