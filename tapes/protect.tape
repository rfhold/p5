# Protect tape: protect and unprotect resources
# Demonstrates: protect.md

Output docs/assets/protect.gif

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

# Start p5 in stack view
Type "p5 -C programs/simple"
Enter
Sleep 500ms

# Wait for resources to load
Wait+Screen /resources/
Sleep 2s

# Use filter to find a specific resource (more reliable than navigation)
Type "/"
Sleep 500ms
Type "my-pet"
Sleep 500ms
Enter
Sleep 1s

# Press P to protect the resource (executes immediately)
Type "P"

# Wait for Protected indicator to appear
Wait+Screen /Protected/
Sleep 2s

# Press P again to unprotect (shows confirmation modal)
Type "P"

# Wait for confirmation modal
Wait+Screen /Unprotect Resource/
Sleep 2s

# Cancel the unprotect
Type "n"
Sleep 1s

# Press P again to show unprotect flow
Type "P"
Wait+Screen /Unprotect Resource/
Sleep 2s

# Confirm the unprotect
Type "y"
Sleep 2s

# Quit
Type "q"
Sleep 1s
