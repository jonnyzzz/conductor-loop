# Installation Guide

This guide covers installing Conductor Loop on macOS, Linux, and Windows.

## Prerequisites

Before installing Conductor Loop, ensure you have:

- **Go 1.21+**: [Download Go](https://go.dev/dl/) (required for source builds)
- **Git**: Any recent version
- **Docker** (optional): For containerized deployment
- **API Tokens**: For your chosen AI agents (Claude, Codex, Gemini, etc.)

### Verify Prerequisites

```bash
# Check Go version
go version  # Should be 1.21 or higher

# Check Git
git --version

# Check Docker (optional)
docker --version
docker-compose --version
```

## Installation Methods

### Option 1: Install/Update `run-agent` from Latest Release

```bash
curl -fsSL https://run-agent.jonnyzzz.com/install.sh | bash
```

The installer:
- Detects your OS/arch (Linux/macOS)
- Downloads from `https://run-agent.jonnyzzz.com/releases/latest/download` first
- Falls back to GitHub release assets if mirror download fails
- Installs to `/usr/local/bin/run-agent` by default

Optional overrides:

```bash
RUN_AGENT_INSTALL_DIR="$HOME/.local/bin" \
RUN_AGENT_DOWNLOAD_BASE="https://run-agent.jonnyzzz.com/releases/latest/download" \
RUN_AGENT_FALLBACK_DOWNLOAD_BASE="https://github.com/jonnyzzz/conductor-loop/releases/latest/download" \
curl -fsSL https://run-agent.jonnyzzz.com/install.sh | bash
```

#### Maintainer Release Validation

For release candidates, validate installer behavior against concrete artifacts before publishing:

```bash
bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh
```

If the platform asset is missing from `dist/`, the smoke script auto-builds it
from `./cmd/run-agent`. Pass `--no-build` to enforce prebuilt-only validation.

This smoke check verifies:
- install from latest mirror URL
- update when latest asset changes
- fallback to secondary download base
- URL normalization for `/releases`, `/releases/download`, and `/releases/latest/download`

Current expectation: `install.sh` does not verify checksums yet; it relies on trusted HTTPS endpoints.

### Option 2: Build from Source

#### 1. Clone the Repository

```bash
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
```

#### 2. Build the Binaries

```bash
# Build the main conductor server
go build -o conductor ./cmd/conductor

# Build the run-agent binary
go build -o run-agent ./cmd/run-agent

# Verify the build
./conductor version
./run-agent --version
```

#### 3. Install System-Wide (Optional)

```bash
# macOS/Linux
sudo mv conductor /usr/local/bin/
sudo mv run-agent /usr/local/bin/

# Or add to PATH
export PATH=$PATH:$(pwd)
```

#### 4. Set Up Configuration

```bash
# Create configuration directory
mkdir -p ~/.conductor

# Create a basic config file
cat > ~/.conductor/config.yaml <<EOF
agents:
  codex:
    type: codex
    token_file: ~/.conductor/tokens/codex.token
    timeout: 300
  claude:
    type: claude
    token_file: ~/.conductor/tokens/claude.token
    timeout: 300

defaults:
  agent: codex
  timeout: 300

api:
  host: 0.0.0.0
  port: 14355
  cors_origins:
    - http://localhost:3000

storage:
  runs_dir: ~/.conductor/runs
EOF

# Create tokens directory
mkdir -p ~/.conductor/tokens

# Add your API tokens (replace with your actual tokens)
echo "your-codex-token-here" > ~/.conductor/tokens/codex.token
echo "your-claude-token-here" > ~/.conductor/tokens/claude.token
chmod 600 ~/.conductor/tokens/*.token
```

### Option 3: Docker Deployment

#### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop

# Create secrets directory
mkdir -p secrets
echo "your-codex-token" > secrets/codex.token
echo "your-claude-token" > secrets/claude.token
chmod 600 secrets/*.token

# Start with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the services
docker-compose down
```

The docker-compose setup includes:
- Conductor server on port 14355
- Frontend on port 3000
- Persistent storage volumes

#### Manual Docker Build

```bash
# Build the Docker image
docker build -t conductor-loop:latest .

# Run the container
docker run -d \
  -p 14355:14355 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/secrets:/secrets \
  -v $(pwd)/runs:/data/runs \
  --name conductor \
  conductor-loop:latest
```

### Option 4: Download Pre-built Binaries Manually

Release binaries are available on the GitHub releases page:

- https://github.com/jonnyzzz/conductor-loop/releases

## Platform-Specific Notes

### macOS

#### Apple Silicon (M1/M2/M3)

Go cross-compilation works natively on Apple Silicon:

```bash
# Build for ARM64 (native)
go build -o conductor ./cmd/conductor

# Build for x86_64 (Rosetta)
GOARCH=amd64 go build -o conductor ./cmd/conductor
```

#### Permissions

macOS may block unsigned binaries. If you see a security warning:

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine conductor
xattr -d com.apple.quarantine run-agent
```

Or: System Preferences → Security & Privacy → Allow anyway

### Linux

#### Common Distributions

Works on all major distributions:
- Ubuntu 20.04+
- Debian 11+
- Fedora 35+
- CentOS 8+
- Arch Linux

#### Systemd Service (Optional)

Create a systemd service for automatic startup:

```bash
sudo tee /etc/systemd/system/conductor.service > /dev/null <<EOF
[Unit]
Description=Conductor Loop Orchestration Server
After=network.target

[Service]
Type=simple
User=conductor
WorkingDirectory=/opt/conductor
ExecStart=/usr/local/bin/conductor --config /etc/conductor/config.yaml --root /opt/conductor
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable conductor
sudo systemctl start conductor
sudo systemctl status conductor
```

### Windows

#### Build on Windows

```powershell
# Using PowerShell
go build -o conductor.exe .\cmd\conductor
go build -o run-agent.exe .\cmd\run-agent

# Run
.\conductor.exe --version
```

#### Windows Paths

Use Windows-style paths in config.yaml:

```yaml
agents:
  codex:
    token_file: C:\Users\YourName\.conductor\tokens\codex.token

storage:
  runs_dir: C:\Users\YourName\.conductor\runs
```

#### Running as a Windows Service (Optional)

Use [NSSM](https://nssm.cc/) to run as a service:

```powershell
# Install NSSM
choco install nssm

# Create service
nssm install Conductor "C:\Path\To\conductor.exe"
nssm set Conductor AppParameters "--config C:\Path\To\config.yaml"
nssm start Conductor
```

## Verifying Installation

### Check Binary Versions

```bash
conductor version
run-agent --version
```

### Optional: Enable Wrapped Shell Aliases (Safe Opt-In)

If you want `claude`, `codex`, and `gemini` terminal calls to be tracked automatically as
Conductor tasks/runs, install shell aliases explicitly:

```bash
run-agent shell-setup install
```

This only edits your shell init file when you run it. It writes a managed block so it can be
removed cleanly later:

```bash
run-agent shell-setup uninstall
```

### Test the Server

```bash
# Start the server
conductor --config ~/.conductor/config.yaml --root $(pwd)

# In another terminal, test the health endpoint
curl http://localhost:14355/api/v1/health

# Expected output:
# {"status":"ok","version":"dev"}
```

### Run a Test Task

```bash
# Create a test task via the API
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "codex",
    "prompt": "Print hello world",
    "working_dir": "/tmp"
  }'

# Check the runs
curl http://localhost:14355/api/v1/runs
```

## Upgrading

### From Source

```bash
cd conductor-loop
git pull origin main
go build -o conductor ./cmd/conductor
go build -o run-agent ./cmd/run-agent
```

### Docker

```bash
# Pull latest image
docker-compose pull

# Restart services
docker-compose up -d
```

## Uninstalling

### Remove Binaries

```bash
# If installed system-wide
sudo rm /usr/local/bin/conductor
sudo rm /usr/local/bin/run-agent

# Or just delete the cloned repository
rm -rf ~/conductor-loop
```

### Remove Configuration and Data

```bash
# Remove config and runs
rm -rf ~/.conductor

# Docker: remove volumes
docker-compose down -v
```

## Troubleshooting Installation Issues

### Go Version Too Old

```bash
# Check version
go version

# Upgrade Go
# macOS: brew upgrade go
# Linux: download from go.dev
# Windows: download from go.dev
```

### Build Errors

```bash
# Clean and rebuild
go clean -cache
go mod tidy
go build -o conductor ./cmd/conductor
```

### Port Already in Use

```bash
# Find process using port 14355
# macOS/Linux:
lsof -i :14355
kill -9 <PID>

# Windows:
netstat -ano | findstr :14355
taskkill /PID <PID> /F

# Or change port in config.yaml
api:
  port: 8081
```

### Permission Denied (Linux)

```bash
# Make binaries executable
chmod +x conductor run-agent

# Fix token file permissions
chmod 600 ~/.conductor/tokens/*.token
```

### Docker Build Fails

```bash
# Clear Docker cache
docker system prune -a

# Rebuild without cache
docker-compose build --no-cache
```

## Next Steps

- [Quick Start Guide](quick-start.md) - Run your first task
- [Configuration Reference](configuration.md) - Configure agents and settings
- [CLI Reference](cli-reference.md) - Learn all commands

## Getting Help

If you encounter issues:

1. Check [Troubleshooting Guide](troubleshooting.md)
2. Search [GitHub Issues](https://github.com/jonnyzzz/conductor-loop/issues)
3. Open a new issue with:
   - Operating system and version
   - Go version (`go version`)
   - Error messages
   - Steps to reproduce
