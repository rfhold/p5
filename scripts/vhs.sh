#!/bin/bash
# Build and run VHS to generate demo GIFs
#
# Usage: ./scripts/vhs.sh [options] [tape...]
#   --build-only   Only build the Docker image
#   --run-only     Skip image build (assumes image exists)
#   --list         List available tapes
#   tape           Tape name(s) to run (without .tape extension)
#                  Use 'all' to run all tapes, 'demo' for main demo
#
# Examples:
#   ./scripts/vhs.sh                    # Build and run demo.tape
#   ./scripts/vhs.sh workflow details   # Run specific tapes
#   ./scripts/vhs.sh all                # Run all tapes
#   ./scripts/vhs.sh --list             # List available tapes

set -e

IMAGE_NAME="p5-vhs"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

build_image() {
    echo "Building VHS Docker image..."
    docker build -f "$PROJECT_ROOT/Dockerfile.vhs" -t "$IMAGE_NAME" "$PROJECT_ROOT"
}

run_tape() {
    local tape="$1"
    echo "Running VHS: $tape"
    docker run --rm -v "$PROJECT_ROOT":/app "$IMAGE_NAME" vhs "$tape"
}

list_tapes() {
    echo "Available tapes:"
    echo "  demo        Main demo (demo.tape)"
    echo ""
    echo "Feature tapes (tapes/):"
    for tape in "$PROJECT_ROOT"/tapes/*.tape; do
        if [[ -f "$tape" ]]; then
            name=$(basename "$tape" .tape)
            echo "  $name"
        fi
    done
}

run_all_tapes() {
    echo "Running all tapes..."
    
    # Run main demo
    run_tape "demo.tape"
    
    # Run all feature tapes
    for tape in "$PROJECT_ROOT"/tapes/*.tape; do
        if [[ -f "$tape" ]]; then
            run_tape "tapes/$(basename "$tape")"
        fi
    done
    
    echo "Done! Generated all GIFs"
}

# Parse arguments
BUILD=true
RUN=true
TAPES=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --build-only)
            RUN=false
            shift
            ;;
        --run-only)
            BUILD=false
            shift
            ;;
        --list)
            list_tapes
            exit 0
            ;;
        -h|--help)
            head -n 15 "$0" | tail -n 14
            exit 0
            ;;
        *)
            TAPES+=("$1")
            shift
            ;;
    esac
done

# Build image if requested
if [[ "$BUILD" == true ]]; then
    build_image
fi

# Run tapes if requested
if [[ "$RUN" == true ]]; then
    if [[ ${#TAPES[@]} -eq 0 ]]; then
        # Default: run demo.tape
        run_tape "demo.tape"
        echo "Done! Generated demo.gif"
    elif [[ "${TAPES[0]}" == "all" ]]; then
        run_all_tapes
    else
        # Run specified tapes
        for tape in "${TAPES[@]}"; do
            if [[ "$tape" == "demo" ]]; then
                run_tape "demo.tape"
            elif [[ -f "$PROJECT_ROOT/tapes/$tape.tape" ]]; then
                run_tape "tapes/$tape.tape"
            elif [[ -f "$PROJECT_ROOT/$tape" ]]; then
                run_tape "$tape"
            elif [[ -f "$PROJECT_ROOT/$tape.tape" ]]; then
                run_tape "$tape.tape"
            else
                echo "Error: Tape not found: $tape" >&2
                echo "Use --list to see available tapes" >&2
                exit 1
            fi
        done
        echo "Done!"
    fi
fi
