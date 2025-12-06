#!/bin/bash
# Build and run VHS to generate demo.gif
#
# Usage: ./scripts/vhs.sh [--build-only|--run-only]
#   --build-only  Only build the Docker image, don't run VHS
#   --run-only    Only run VHS (assumes image already exists)
#   (no args)     Build image and run VHS

set -e

IMAGE_NAME="p5-vhs"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

build_image() {
    echo "Building VHS Docker image..."
    docker build -f "$PROJECT_ROOT/Dockerfile.vhs" -t "$IMAGE_NAME" "$PROJECT_ROOT"
}

run_vhs() {
    echo "Running VHS to generate demo.gif..."
    docker run --rm -v "$PROJECT_ROOT":/app "$IMAGE_NAME"
    echo "Done! Generated demo.gif"
}

case "${1:-}" in
    --build-only)
        build_image
        ;;
    --run-only)
        run_vhs
        ;;
    *)
        build_image
        run_vhs
        ;;
esac
