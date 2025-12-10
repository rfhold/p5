# Stack init tape: create a new stack via stack selector
# Demonstrates: stack-init.md

Output docs/assets/stack-init.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set WaitTimeout 30s

# Setup: clean state - no backend stacks, but stack file exists
Hide
Type "rm -rf programs/simple/.pulumi"
Enter
Sleep 1s
Type "clear"
Enter
Sleep 500ms
Show

# Start p5 - shows stack selector since Pulumi.dev.yaml exists
Type "p5 -C programs/simple"
Enter

# Wait for stack selector
Wait+Screen /Select Stack/
Sleep 2s

# Select "+ New Stack" (first option, already selected)
Enter

# Wait for Initialize Stack modal
Wait+Screen /Initialize Stack/
Sleep 1s

# Enter stack name - use "prod" to differentiate from the existing "dev" file
Type "prod"
Sleep 500ms
Enter
Sleep 1s

# Select passphrase secrets provider (should be first suggestion)
Wait+Screen /Select secrets provider/
Sleep 1s
Enter

# Wait for stack to be created and resources to load
# (passphrase step may be skipped if PULUMI_CONFIG_PASSPHRASE is set in env)
Wait+Screen /resources/
Sleep 3s

# Quit
Type "q"
Sleep 1s
