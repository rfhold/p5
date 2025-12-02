ARG PULUMI_VERSION=3.209.0
ARG RUST_VERSION=1.86.0
ARG PULUMI_IMAGE_SUFFIX=""

FROM docker.cr.holdenitdown.net/pulumi/pulumi${PULUMI_IMAGE_SUFFIX}:${PULUMI_VERSION} AS pulumi

ENV PULUMI_HOME=/root/.pulumi
RUN mkdir -p ${PULUMI_HOME} && chmod 777 ${PULUMI_HOME}

FROM pulumi AS test

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
		apt-get install -y --no-install-recommends \
		ca-certificates \
		build-essential \
		curl && \
		rm -rf /var/lib/apt/lists/*

ARG RUST_VERSION=1.86.0
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --profile minimal --default-toolchain ${RUST_VERSION}
ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /app

RUN mkdir -p src pulumi-automation/src && \
		touch src/main.rs pulumi-automation/src/lib.rs

COPY ./.cargo/config.toml ./.cargo/config.toml
COPY ./Cargo.toml ./Cargo.lock ./
COPY ./pulumi-automation/Cargo.toml ./pulumi-automation/

RUN --mount=type=cache,target=/usr/local/cargo/git/db \
    --mount=type=cache,target=/usr/local/cargo/registry/ \
		cargo fetch --locked

COPY . .

ENTRYPOINT [ "cargo", "test", "--workspace", "--test"]

FROM rust:1.86.0 AS build

WORKDIR /app

RUN mkdir -p src pulumi-automation/src && \
		touch src/main.rs pulumi-automation/src/lib.rs

COPY ./.cargo/config.toml ./.cargo/config.toml
COPY ./Cargo.toml ./Cargo.lock ./
COPY ./pulumi-automation/Cargo.toml ./pulumi-automation/

RUN --mount=type=cache,target=/usr/local/cargo/git/db \
    --mount=type=cache,target=/usr/local/cargo/registry/ \
		cargo fetch --locked

COPY src ./src
COPY pulumi-automation/src ./pulumi-automation/src

RUN --mount=type=cache,target=/app/target/ \
    --mount=type=cache,target=/usr/local/cargo/git/db \
    --mount=type=cache,target=/usr/local/cargo/registry/ \
		cargo build --release --bin p5 && cp target/release/p5 /app/p5

FROM pulumi AS vhs

ARG VHS_VERSION=v0.9.0
RUN go install github.com/charmbracelet/vhs@${VHS_VERSION}

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
		apt-get install -y --no-install-recommends \
		chromium \
		ffmpeg && \
		rm -rf /var/lib/apt/lists/*

COPY --from=tsl0922/ttyd:alpine /usr/bin/ttyd /usr/bin/ttyd

WORKDIR /app

COPY --from=build /app/p5 /usr/bin/p5

COPY . .

WORKDIR /app/tapes

ENV PULUMI_CONFIG_PASSPHRASE_FILE="/app/tapes/passphrase.txt"
ENV PULUMI_BACKEND_URL="file:///app/tapes"

ENV VHS_NO_SANDBOX=true

ENTRYPOINT [ "/bin/bash", "-c", "vhs demo.tape" ]
