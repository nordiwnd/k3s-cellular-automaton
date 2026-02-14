# syntax=docker/dockerfile:1.4
FROM tonistiigi/xx AS xx

FROM --platform=$BUILDPLATFORM rust:1.82-alpine AS builder

# Copy xx helpers
COPY --from=xx / /

# Install build dependencies
RUN apk add --no-cache clang lld musl-dev git file

# Configure xx for the target platform
ARG TARGETPLATFORM
RUN xx-apk add --no-cache musl-dev gcc

WORKDIR /app

# Dependency caching layer
COPY Cargo.toml ./
# Create dummy main.rs to build dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/app/target \
    xx-cargo build --release --target-dir ./target && \
    rm src/main.rs

# Build actual application
COPY src ./src
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/app/target \
    xx-cargo build --release --target-dir ./target && \
    find ./target -name cells-worker -type f -exec cp {} /cells-worker \;

FROM scratch
COPY --from=builder /cells-worker /cells-worker
ENTRYPOINT ["/cells-worker"]
