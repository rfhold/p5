# VHS Demo Script for p5
# Run with: vhs demo.tape

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set WaitTimeout 5s

# Clean any existing state and initialize a fresh stack
Hide
Type "rm -rf programs/simple/.pulumi && cd programs/simple && pulumi stack init dev && cd /app"
Enter
Sleep 2000ms
Type "clear"
Enter
Sleep 500ms
Show

# Run p5 with up command directly (shows preview first)
Type "p5 -C programs/simple up"
Enter

# Wait for preview to complete (shows [+] for creates)
Wait+Screen /\[.\]/
Sleep 1000ms

# Navigate down a few resources
Down@200ms 3
Sleep 500ms

# Open details view
Type "D"
Sleep 2000ms

# Go back to resource list
Escape
Sleep 500ms

# Execute the operation with ctrl+u
Ctrl+u

# Wait for up operation to complete
Wait+Screen /p5-test-project-dev/
Sleep 2000ms

# Show details of the stack
Type "D"
Wait+Screen /p5 test project!/
Sleep 2000ms

# Go back to resource list
Escape
Escape
Wait+Screen /resources/
Sleep 2000ms

# Destroy the stack
Ctrl+d

# Wait for destroy confirmation modal
Wait+Screen /Execute Destroy/
Sleep 1000ms

# Confirm destroy
Type "y"

# Wait for destroy to complete
Wait+Screen /deleted/
Sleep 4000ms
