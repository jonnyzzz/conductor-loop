FROM golang:1.21-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /conductor ./cmd/conductor
RUN go build -o /run-agent ./cmd/run-agent

FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache bash curl jq

# Copy binaries
COPY --from=builder /conductor /usr/local/bin/
COPY --from=builder /run-agent /usr/local/bin/

# Create directories
RUN mkdir -p /data/runs /data/config

WORKDIR /data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/v1/health || exit 1

CMD ["conductor"]
