//go:build integration

package main

// Integration tests are split across multiple files:
// - integration_helpers_test.go: Shared test infrastructure
// - integration_init_test.go: Initialization flow tests
// - integration_preview_test.go: Preview & execute tests
// - integration_stack_test.go: Stack view tests
// - integration_flags_test.go: Resource flags tests
// - integration_help_test.go: Help dialog tests
