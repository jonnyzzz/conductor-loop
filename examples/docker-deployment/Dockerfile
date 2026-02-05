# Multi-stage build for Conductor Loop

# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o conductor \
    ./cmd/conductor

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o run-agent \
    ./cmd/run-agent

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    curl \
    bash \
    tzdata

# Create non-root user
RUN addgroup -g 1000 conductor && \
    adduser -D -u 1000 -G conductor conductor

# Create directories
RUN mkdir -p /data/runs /config /secrets && \
    chown -R conductor:conductor /data /config

WORKDIR /app

# Copy binaries from builder
COPY --from=builder --chown=conductor:conductor /app/conductor /app/run-agent ./

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Switch to non-root user
USER conductor

# Expose API port
EXPOSE 8080

# Run conductor
CMD ["./conductor", "--config", "/config/config.yaml", "serve"]
