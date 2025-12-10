# Main workflow tape: up, refresh, destroy (preview + execute for each)
# Demonstrates: preview.md, execute.md

Output docs/assets/workflow.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set WaitTimeout 30s

# Setup: clean state
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && cd /app"
Enter
Sleep 2s
Type "clear"
Enter
Sleep 500ms
Show

# Start p5 with up preview
Type "p5 -C programs/simple up"
Enter

# Wait for up preview done
Wait+Screen /done/
Sleep 2s

# Execute up - wait for "created" status on resources
Ctrl+u
Sleep 5s
Wait+Screen /created/
Sleep 2s

# Refresh preview
Type "r"
Wait+Screen /done/
Sleep 2s

# Execute refresh - wait for "refreshed" status
Ctrl+r
Wait+Screen /refreshed/
Sleep 2s

# Destroy preview
Type "d"
Wait+Screen /done/
Sleep 2s

# Execute destroy - wait for "deleted" status
Ctrl+d
Wait+Screen /deleted/
Sleep 3s
