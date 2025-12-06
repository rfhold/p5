# Dockerfile for recording VHS demos with Pulumi and Go
# Optimized for layer caching

# =============================================================================
# Stage 1: Base image with all system dependencies (rarely changes)
# =============================================================================
FROM pulumi/pulumi-go:latest AS base

# Install VHS dependencies (ffmpeg, chromium)
RUN apt-get update && apt-get install -y \
    ffmpeg \
    chromium \
    && rm -rf /var/lib/apt/lists/*

# Copy ttyd from alpine image
COPY --from=tsl0922/ttyd:alpine /usr/bin/ttyd /usr/bin/ttyd

# Install VHS (pinned version for reproducibility)
ARG VHS_VERSION=v0.9.0
RUN go install github.com/charmbracelet/vhs@${VHS_VERSION}

# =============================================================================
# Stage 2: Download p5 Go dependencies (changes when go.mod/go.sum change)
# =============================================================================
FROM base AS deps

WORKDIR /app

# Copy only dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# =============================================================================
# Stage 3: Build p5 binary (changes when source code changes)
# =============================================================================
FROM deps AS build

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build p5
RUN go build -o /usr/bin/p5 ./cmd/p5

# =============================================================================
# Stage 4: Prepare test project dependencies (cache Pulumi provider downloads)
# =============================================================================
FROM build AS test-deps

# Copy test project
COPY programs/simple/ ./programs/simple/

# Pre-download Pulumi providers and pre-compile the Go program
WORKDIR /app/programs/simple

# Environment for Pulumi
ENV PULUMI_BACKEND_URL=file://.
ENV PULUMI_CONFIG_PASSPHRASE=""
ENV GOFLAGS=-buildvcs=false

# Download Go dependencies for the test project
RUN go mod download

# Pre-compile the Pulumi program (this caches the build)
RUN go build -o /tmp/pulumi-program .

# Initialize stack and run preview to download providers
RUN pulumi stack init demo --non-interactive || true && \
    pulumi preview --non-interactive 2>/dev/null || true && \
    rm -rf .pulumi

# =============================================================================
# Stage 5: Final image with everything ready
# =============================================================================
FROM base AS final

# Environment variables
ENV VHS_NO_SANDBOX=true
ENV PULUMI_BACKEND_URL=file://.
ENV PULUMI_CONFIG_PASSPHRASE=""
ENV GOFLAGS=-buildvcs=false

WORKDIR /app

# Copy built p5 binary
COPY --from=build /usr/bin/p5 /usr/bin/p5

# Copy Go module cache from deps stage (includes test project deps)
COPY --from=test-deps /go/pkg/mod /go/pkg/mod

# Copy Go build cache (pre-compiled test program dependencies)
COPY --from=test-deps /root/.cache/go-build /root/.cache/go-build

# Copy Pulumi plugins from test-deps stage
COPY --from=test-deps /root/.pulumi/plugins /root/.pulumi/plugins

# Copy everything else at runtime via volume mount
# The demo.tape and test projects will be mounted

CMD ["vhs", "demo.tape"]
