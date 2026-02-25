# Docker Deployment Example

Production-ready Docker deployment with Docker Compose, including frontend, reverse proxy, and health monitoring.

## What This Example Demonstrates

- Containerized Conductor Loop deployment
- Multi-container setup with Docker Compose
- Reverse proxy configuration (nginx)
- Volume management for persistent data
- Environment variable configuration
- Health checks and monitoring
- Production best practices

## Architecture

```
┌─────────────────────────────────────────┐
│           nginx (reverse proxy)          │
│         :443 (HTTPS) / :80 (HTTP)       │
└─────────────┬───────────────────────────┘
              │
      ┌───────┴───────┐
      │               │
┌─────▼──────┐ ┌─────▼──────┐
│ conductor  │ │  frontend  │
│   :14355    │ │   (static) │
└─────┬──────┘ └────────────┘
      │
┌─────▼──────────────────────┐
│  Volumes                   │
│  - conductor-runs          │
│  - conductor-secrets       │
└────────────────────────────┘
```

## Files in This Example

- `README.md` - This file
- `docker-compose.yml` - Multi-container orchestration
- `Dockerfile` - Conductor Loop container image
- `Dockerfile.frontend` - Frontend container image
- `nginx.conf` - Reverse proxy configuration
- `config.yaml` - Production configuration
- `.env.example` - Environment variables template
- `deploy.sh` - Deployment script
- `backup.sh` - Backup script

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- SSL certificates (for HTTPS)

## Quick Start

### 1. Clone and Configure

```bash
cd examples/docker-deployment

# Create environment file
cp .env.example .env
nano .env  # Edit with your values
```

### 2. Deploy

```bash
./deploy.sh
```

This will:
- Build Docker images
- Create volumes
- Start all containers
- Run health checks
- Display status

### 3. Verify

```bash
# Check container status
docker-compose ps

# Check health
curl http://localhost/api/v1/health

# View logs
docker-compose logs -f conductor
```

### 4. Access

- API: `http://localhost/api/`
- Web UI: `http://localhost/`
- For HTTPS, configure SSL certificates in nginx.conf

## Configuration

### Environment Variables (.env)

```bash
# Conductor Configuration
CONDUCTOR_VERSION=latest
CONDUCTOR_API_PORT=14355

# Agent API Keys (use secrets manager in production)
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
GEMINI_API_KEY=...

# Storage
JRUN_RUNS_DIR=/data/runs
SECRETS_DIR=/secrets

# Networking
EXTERNAL_PORT=80
EXTERNAL_HTTPS_PORT=443
DOMAIN=conductor.example.com
```

### Volume Mounts

**conductor-runs** (persistent):
- Stores all run data
- Must be backed up regularly
- Path: `/data/runs`

**conductor-secrets** (read-only):
- Stores API keys
- Restricted permissions
- Path: `/secrets`

## Docker Compose Services

### conductor

**Image:** Built from Dockerfile
**Purpose:** Main Conductor Loop API server
**Ports:** 14355 (internal only)
**Volumes:**
- conductor-runs:/data/runs
- conductor-secrets:/secrets:ro

**Health Check:**
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:14355/api/v1/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

### nginx

**Image:** nginx:alpine
**Purpose:** Reverse proxy and static file serving
**Ports:**
- 80:80 (HTTP)
- 443:443 (HTTPS)

**Configuration:** nginx.conf

### frontend

**Image:** Built from Dockerfile.frontend
**Purpose:** React web UI
**Served by:** nginx

## Production Dockerfile

```dockerfile
# Multi-stage build for smaller image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o conductor ./cmd/conductor

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/conductor .

# Expose API port
EXPOSE 14355

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD curl -f http://localhost:14355/api/v1/health || exit 1

# Run as non-root user
RUN adduser -D -u 1000 conductor
USER conductor

CMD ["./conductor", "--config", "/config/config.yaml", "serve"]
```

**Benefits:**
- Multi-stage build (smaller image)
- Non-root user (security)
- Health check (monitoring)
- CA certificates (HTTPS support)

## Nginx Configuration

```nginx
upstream conductor_api {
    server conductor:14355;
}

server {
    listen 80;
    server_name conductor.example.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name conductor.example.com;

    # SSL Configuration
    ssl_certificate /etc/nginx/ssl/conductor.crt;
    ssl_certificate_key /etc/nginx/ssl/conductor.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # API endpoints
    location /api/ {
        proxy_pass http://conductor_api;
        proxy_http_version 1.1;

        # SSE support
        proxy_set_header Connection '';
        proxy_set_header Upgrade $http_upgrade;
        proxy_buffering off;
        proxy_cache off;

        # Headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts for long-running requests
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }

    # Frontend
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;

        # Cache static assets
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
```

**Features:**
- HTTPS with TLS 1.2/1.3
- HTTP to HTTPS redirect
- SSE streaming support
- Static asset caching
- Proper timeout handling
- Health check endpoint

## Deployment Script

```bash
#!/bin/bash
set -e

echo "====================================="
echo "Conductor Loop - Docker Deployment"
echo "====================================="

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Docker required"; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose required"; exit 1; }

# Load environment
if [ ! -f .env ]; then
    echo "Error: .env file not found"
    echo "Copy .env.example to .env and configure"
    exit 1
fi

source .env

# Build images
echo "Building Docker images..."
docker-compose build --no-cache

# Create volumes
echo "Creating volumes..."
docker volume create conductor-runs
docker volume create conductor-secrets

# Start services
echo "Starting services..."
docker-compose up -d

# Wait for services to be healthy
echo "Waiting for services to start..."
sleep 10

# Health check
echo "Running health checks..."
for i in {1..30}; do
    if curl -sf http://localhost/api/v1/health > /dev/null; then
        echo "✓ Conductor API healthy"
        break
    fi
    sleep 2
done

# Display status
echo ""
echo "====================================="
echo "Deployment Complete!"
echo "====================================="
docker-compose ps
echo ""
echo "Access Conductor Loop at:"
echo "  API: http://localhost/api/"
echo "  Web UI: http://localhost/"
echo ""
echo "View logs with: docker-compose logs -f"
```

## Backup Strategy

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups/conductor"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

echo "Backing up Conductor Loop data..."

# Backup volumes
docker run --rm \
  -v conductor-runs:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/runs-$TIMESTAMP.tar.gz /data

docker run --rm \
  -v conductor-secrets:/secrets \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/secrets-$TIMESTAMP.tar.gz /secrets

# Keep last 30 days
find $BACKUP_DIR -name "*.tar.gz" -mtime +30 -delete

echo "Backup complete: $BACKUP_DIR"
```

**Cron schedule:**
```bash
# Daily at 2 AM
0 2 * * * /opt/conductor/backup.sh >> /var/log/conductor-backup.log 2>&1
```

## Scaling

### Horizontal Scaling

Run multiple conductor instances behind load balancer:

```yaml
# docker-compose.scale.yml
services:
  conductor:
    deploy:
      replicas: 3

  nginx:
    depends_on:
      - conductor
```

Deploy:
```bash
docker-compose -f docker-compose.yml -f docker-compose.scale.yml up -d --scale conductor=3
```

### Resource Limits

```yaml
services:
  conductor:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

## Monitoring

### Prometheus Metrics

Add Prometheus exporter:

```yaml
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    ports:
      - "9090:9090"
```

### Log Aggregation

Use Docker logging drivers:

```yaml
services:
  conductor:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

Or send to external system:

```yaml
logging:
  driver: "syslog"
  options:
    syslog-address: "tcp://logs.example.com:514"
```

## Security Best Practices

### 1. Use Secrets Management

**Docker Swarm secrets:**
```yaml
secrets:
  anthropic_key:
    external: true

services:
  conductor:
    secrets:
      - anthropic_key
```

### 2. Network Isolation

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true

services:
  nginx:
    networks:
      - frontend
  conductor:
    networks:
      - frontend
      - backend
```

### 3. Read-Only Filesystem

```yaml
services:
  conductor:
    read_only: true
    tmpfs:
      - /tmp
      - /var/run
```

### 4. Security Scanning

```bash
# Scan images for vulnerabilities
docker scan conductor:latest

# Use Trivy
trivy image conductor:latest
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs conductor

# Check configuration
docker-compose config

# Verify volumes
docker volume ls
docker volume inspect conductor-runs
```

### Health Check Failing

```bash
# Test from host
curl http://localhost/api/v1/health

# Test from within container
docker-compose exec conductor curl http://localhost:14355/api/v1/health

# Check conductor logs
docker-compose logs conductor | grep ERROR
```

### Permission Denied Errors

```bash
# Fix volume permissions
docker-compose exec conductor chown -R conductor:conductor /data/runs
```

### Out of Disk Space

```bash
# Check disk usage
df -h

# Clean up old runs
docker-compose exec conductor find /data/runs -mtime +30 -delete

# Prune Docker resources
docker system prune -a
```

## Maintenance

### Update Conductor

```bash
# Pull latest images
docker-compose pull

# Recreate containers
docker-compose up -d

# Verify health
curl http://localhost/api/v1/health
```

### Backup Before Update

```bash
./backup.sh
# Then update
```

### Rollback

```bash
# Use specific version
CONDUCTOR_VERSION=v1.0.0 docker-compose up -d
```

## Next Steps

After deploying with Docker:

1. Set up SSL certificates (Let's Encrypt recommended)
2. Configure monitoring and alerting
3. Implement backup automation
4. Set up log aggregation
5. Create disaster recovery plan
6. Review [Best Practices](../best-practices.md)

## Production Checklist

- [ ] SSL certificates configured
- [ ] Environment variables secured
- [ ] Volumes backed up regularly
- [ ] Health checks tested
- [ ] Resource limits configured
- [ ] Monitoring set up
- [ ] Log aggregation enabled
- [ ] Firewall rules configured
- [ ] Security scan passed
- [ ] Rollback procedure tested
- [ ] Documentation updated
- [ ] Team trained on operations

## Related Examples

- [config templates](../configs/config.docker.yaml) - Docker-specific config
- [best-practices](../best-practices.md) - Production guidelines
