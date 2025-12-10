# Resource targeting tape: target, replace, exclude flags
# Demonstrates: resource-targetting.md

Output docs/assets/targeting.gif

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

# Start p5 - will show replace operations
Type "p5 -C programs/simple"
Enter
Sleep 500ms

# Wait for preview
Wait+Screen /resources/
Sleep 1s

# Navigate to first resource
Down@200ms 1
Sleep 500ms

# Mark as target
Type "T"
Sleep 1s

# Navigate down and mark another
Down@200ms 1
Sleep 500ms
Type "T"
Sleep 1s

# Navigate down and mark as exclude
Down@200ms 1
Sleep 500ms
Type "E"
Sleep 1s

# Navigate and mark as replace
Down@200ms 1
Sleep 500ms
Type "R"
Sleep 1s

# Show visual mode
Up@200ms 2
Sleep 500ms
Type "v"
Sleep 500ms
Down@200ms 2
Sleep 500ms
Type "E"
Sleep 1s
Escape
Sleep 500ms

# Clear all flags
Type "C"
Sleep 1s

# Quit
Type "q"
Sleep 1s
