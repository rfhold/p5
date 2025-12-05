#!/bin/bash
# Run tests with coverage and generate reports
#
# Usage: ./scripts/coverage.sh [options]
#   -html     Open HTML report in browser
#   -func     Show per-function breakdown (default: summary only)
#   -v        Verbose test output

set -e

# Parse arguments
OPEN_HTML=false
SHOW_FUNC=false
VERBOSE=""

for arg in "$@"; do
    case "$arg" in
        -html) OPEN_HTML=true ;;
        -func) SHOW_FUNC=true ;;
        -v) VERBOSE="-v" ;;
    esac
done

# Patterns to exclude from coverage (pipe-separated regex)
EXCLUDE_PATTERNS="fakes\.go|_mock\.go|_test\.go|/proto/|/testdata/"

# Find packages, excluding proto, test/example programs, and testdata directories
PACKAGES=$(go list ./... | grep -v -E '/proto$|/testdata$|^github.com/rfhold/p5/test/|^github.com/rfhold/p5/programs/')

echo "Running tests with coverage (including integration tests)..."
go test $VERBOSE -tags=integration -covermode=count -coverprofile=coverage.out.tmp $PACKAGES

# Filter out excluded files
if grep -v -E "$EXCLUDE_PATTERNS" coverage.out.tmp > coverage.out 2>/dev/null; then
    rm coverage.out.tmp
else
    # If grep found nothing to exclude, use the original
    mv coverage.out.tmp coverage.out
fi

echo ""
echo "=== Coverage Summary ==="

if [ "$SHOW_FUNC" = true ]; then
    go tool cover -func=coverage.out
    echo ""
fi

# Extract and display total coverage
TOTAL=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo "Total Coverage: $TOTAL"

# Open HTML report if requested
if [ "$OPEN_HTML" = true ]; then
    echo ""
    echo "Opening HTML report..."
    go tool cover -html=coverage.out
fi
