# Stage 1: Build go-launcher
FROM golang:latest AS go-builder

WORKDIR /build

# Copy go-launcher source code
COPY go-launcher/ .

# Download dependencies and build
RUN go mod download && \
    go mod verify && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o launcher .

# Stage 2: Build rotel
FROM rust:latest AS rust-builder

WORKDIR /build

# Install git and clone rotel repository
RUN apt-get update && \
    apt-get install -y git cmake openssl protobuf-compiler libzstd-dev libclang-dev && \
    rm -rf /var/lib/apt/lists/*

ARG ROTEL_SHA=fluent
# Clone rotel and checkout fluent branch
RUN git clone https://github.com/streamfold/rotel.git . && \
    git checkout ${ROTEL_SHA}

# Build release version of rotel
RUN cargo build --features fluent_receiver --release

# Stage 3: Final minimal image
FROM debian:trixie-slim

WORKDIR /app

# Install minimal runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy built binaries from previous stages
COPY --from=go-builder /build/launcher /usr/local/bin/launcher
COPY --from=rust-builder /build/target/release/rotel /usr/local/bin/rotel

# Create entrypoint script
COPY scripts/entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
