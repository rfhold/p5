# Details tape: resource details panel and diff view
# Demonstrates: details.md

Output docs/assets/details.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# Setup: create resources first
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && pulumi up -y && cd /app"
Enter
Sleep 10s
Type "clear"
Enter
Sleep 500ms
Show

# Start p5 - will show replace operations due to timestamp triggers
Type "p5 -C programs/simple"
Enter
Sleep 500ms

# Wait for preview
Wait+Screen /resources/
Sleep 1s

# Navigate to a resource with changes
Down@200ms 2
Sleep 500ms

# Open details panel
Type "D"
Sleep 2s

# Navigate within details
Down@200ms 3
Sleep 1s

# Close details
Escape
Sleep 500ms

# Navigate to another resource
Down@200ms 1
Sleep 500ms

# Open details again
Type "D"
Sleep 2s

# Close and quit
Escape
Sleep 500ms
Type "q"
Sleep 1s
