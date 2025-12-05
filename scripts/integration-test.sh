#!/bin/bash
set -e

# Integration test runner for p5
# Runs both Bubble Tea TUI tests and Pulumi Automation API tests

# Set passphrase for Pulumi local backend (no cloud deps needed)
export PULUMI_CONFIG_PASSPHRASE="test-passphrase-12345"

# Force ASCII terminal for reproducible golden files
export TERM=dumb

echo "=== Running Pulumi Integration Tests ==="
go test -tags=integration ./internal/pulumi -v -timeout=10m "$@"

echo ""
echo "=== Running Bubble Tea Integration Tests ==="
go test -tags=integration ./cmd/p5 -v -timeout=5m "$@"

echo ""
echo "=== All Integration Tests Passed ==="
