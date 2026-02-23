FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /run-agent ./cmd/run-agent

FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache bash curl jq

# Copy binaries
COPY --from=builder /run-agent /usr/local/bin/

# Create directories
RUN mkdir -p /data/runs /data/config

WORKDIR /data

EXPOSE 14355

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:14355/api/v1/health || exit 1

CMD ["run-agent", "serve"]
