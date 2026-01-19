# PerpDEX Node Dockerfile
# Multi-stage build for optimized image size

# ============ Build Stage ============
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    linux-headers

# Set working directory
WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always)" \
    -o /perpdexd ./cmd/perpdexd

# ============ Runtime Stage ============
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    jq \
    bash

# Create non-root user
RUN addgroup -S perpdex && adduser -S perpdex -G perpdex

# Copy binary from builder
COPY --from=builder /perpdexd /usr/local/bin/perpdexd

# Copy entrypoint script
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create data directory
RUN mkdir -p /root/.perpdex && chown -R perpdex:perpdex /root/.perpdex

# Expose ports
# P2P
EXPOSE 26656
# RPC
EXPOSE 26657
# gRPC
EXPOSE 9090
# REST API
EXPOSE 1317
# Prometheus metrics
EXPOSE 26660

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:26657/health || exit 1

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]

# Default command
CMD ["start"]
