# Development Environment Setup

This guide walks you through setting up a development environment for contributing to conductor-loop.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Cloning the Repository](#cloning-the-repository)
3. [Installing Dependencies](#installing-dependencies)
4. [Building from Source](#building-from-source)
5. [Running Locally](#running-locally)
6. [Hot Reload for Development](#hot-reload-for-development)
7. [Debugging Techniques](#debugging-techniques)
8. [IDE Setup](#ide-setup)
9. [Docker Development](#docker-development)
10. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Tools

**Go:**
- Version: 1.21 or higher
- Installation: https://go.dev/doc/install

```bash
# Verify installation
go version
# Should output: go version go1.21.x ...
```

**Node.js & npm:**
- Version: Node.js 18+ and npm 9+
- Installation: https://nodejs.org/

```bash
# Verify installation
node --version  # Should be v18.x or higher
npm --version   # Should be 9.x or higher
```

**Git:**
- Version: 2.x or higher
- Installation: https://git-scm.com/

```bash
# Verify installation
git --version
```

### Optional Tools

**golangci-lint:**
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Verify installation
golangci-lint --version
```

**Docker:**
```bash
# For running integration tests and containerized builds
docker --version
docker-compose --version
```

**Make:**
```bash
# Most Unix systems have make installed
make --version
```

---

## Cloning the Repository

### Clone via HTTPS

```bash
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
```

### Clone via SSH

```bash
git clone git@github.com:jonnyzzz/conductor-loop.git
cd conductor-loop
```

### Verify Repository

```bash
# Check repository structure
ls -la

# Expected output:
# cmd/          - Entry points
# internal/     - Private packages
# frontend/     - React UI
# docs/         - Documentation
# tests/        - Integration tests
# go.mod        - Go dependencies
# package.json  - Frontend dependencies
# README.md
```

---

## Installing Dependencies

### Backend Dependencies

```bash
# Install Go dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy up (remove unused dependencies)
go mod tidy
```

### Frontend Dependencies

```bash
# Navigate to frontend directory
cd frontend

# Install npm dependencies
npm install

# Or use npm ci for clean install
npm ci

# Return to root
cd ..
```

---

## Building from Source

### Build Backend

```bash
# Build all binaries
go build -o bin/ ./cmd/...

# Build specific binary
go build -o bin/conductor ./cmd/conductor
go build -o bin/run-agent ./cmd/run-agent

# Verify binaries
ls -la bin/
./bin/conductor --version
./bin/run-agent --version
```

### Build Frontend

```bash
cd frontend

# Development build
npm run dev

# Production build
npm run build

# Output: frontend/dist/

cd ..
```

### Build Everything

```bash
# Using Makefile (if available)
make build

# Or manually
go build -o bin/ ./cmd/...
cd frontend && npm run build && cd ..
```

---

## Running Locally

### Setup Configuration

```bash
# Create config directory
mkdir -p ~/.conductor

# Create configuration file
cat > ~/.conductor/config.yaml <<EOF
agents:
  claude:
    type: claude
    token_file: ~/.claude/token

defaults:
  agent: claude
  timeout: 3600

api:
  host: 0.0.0.0
  port: 8080

storage:
  runs_dir: ~/run-agent
EOF

# Create token file (replace with your actual token)
mkdir -p ~/.claude
echo "your-api-token-here" > ~/.claude/token
chmod 600 ~/.claude/token
```

### Run API Server

```bash
# Run from source
go run ./cmd/conductor serve --config ~/.conductor/config.yaml

# Or use built binary
./bin/conductor serve --config ~/.conductor/config.yaml

# Server should start on http://localhost:8080
```

### Run Frontend (Development Mode)

```bash
cd frontend

# Start development server with hot reload
npm run dev

# Frontend should start on http://localhost:5173
# Proxies API requests to http://localhost:8080
```

### Run Task Manually

```bash
# Using CLI
./bin/run-agent \
  --config ~/.conductor/config.yaml \
  --project my-project \
  --task task-20260220-140000-my-task \
  --agent claude \
  --prompt "Your task prompt here"

# Check results
ls ~/run-agent/my-project/task-20260220-140000-my-task/runs/
```

---

## Hot Reload for Development

### Backend Hot Reload

**Using Air (recommended):**

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Create .air.toml config
cat > .air.toml <<EOF
root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/conductor"
  cmd = "go build -o ./tmp/conductor ./cmd/conductor"
  delay = 1000
  exclude_dir = ["tmp", "vendor", "frontend"]
  include_ext = ["go"]
  stop_on_error = true

[log]
  time = true
EOF

# Run with hot reload
air
```

**Using fswatch (alternative):**

```bash
# Install fswatch (macOS)
brew install fswatch

# Watch and rebuild on changes
fswatch -o internal/ cmd/ | xargs -n1 -I{} sh -c 'go build -o bin/conductor ./cmd/conductor && ./bin/conductor serve'
```

### Frontend Hot Reload

```bash
cd frontend

# Vite has built-in hot reload
npm run dev

# Changes to src/ will automatically reload the browser
```

---

## Debugging Techniques

### Backend Debugging

**Using Delve:**

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug main executable
dlv debug ./cmd/conductor -- serve --config ~/.conductor/config.yaml

# In delve prompt:
(dlv) break main.main
(dlv) continue
(dlv) next
(dlv) print varName
(dlv) quit
```

**Using VS Code:**

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Conductor",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/conductor",
      "args": ["serve", "--config", "${env:HOME}/.conductor/config.yaml"],
      "env": {}
    }
  ]
}
```

**Print Debugging:**

```go
import "log"

log.Printf("Debug: value=%v", value)
log.Printf("Debug: entering function %s", funcName)
```

### Frontend Debugging

**Browser DevTools:**

```bash
# Start dev server
cd frontend && npm run dev

# Open browser: http://localhost:5173
# Press F12 to open DevTools
# Use Console, Network, and React DevTools tabs
```

**VS Code Debugging:**

Install "Debugger for Chrome" extension, then add to `.vscode/launch.json`:

```json
{
  "name": "Launch Frontend",
  "type": "chrome",
  "request": "launch",
  "url": "http://localhost:5173",
  "webRoot": "${workspaceFolder}/frontend/src"
}
```

### Logging

**Backend:**

```go
import "log"

// Set up logging
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Log messages
log.Println("Info message")
log.Printf("Debug: %v", value)
```

**Frontend:**

```typescript
// Console logging
console.log('Info:', data)
console.error('Error:', error)
console.debug('Debug:', value)
```

---

## IDE Setup

### VS Code

**Recommended Extensions:**

```json
{
  "recommendations": [
    "golang.go",
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss",
    "ms-vscode.vscode-typescript-next"
  ]
}
```

**Settings (.vscode/settings.json):**

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "[typescriptreact]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

### GoLand / IntelliJ IDEA

**Setup:**

1. Open project in GoLand
2. GoLand will auto-detect go.mod and download dependencies
3. Configure Go SDK: Preferences → Go → GOROOT
4. Enable gofmt on save: Preferences → Tools → File Watchers
5. Install golangci-lint integration: Preferences → Tools → golangci-lint

**Run Configuration:**

```
Name: Run Conductor
Kind: Go Build
Package: ./cmd/conductor
Program arguments: serve --config $HOME/.conductor/config.yaml
```

### Vim/Neovim

**Install vim-go:**

```vim
" In .vimrc or init.vim
Plug 'fatih/vim-go', { 'do': ':GoUpdateBinaries' }

" Configure
let g:go_fmt_command = "goimports"
let g:go_auto_type_info = 1
let g:go_metalinter_command = "golangci-lint"
```

---

## Docker Development

### Build Docker Image

```bash
# Build image
docker build -t conductor-loop:dev .

# Verify image
docker images | grep conductor-loop
```

### Run in Docker

```bash
# Run conductor server
docker run -p 8080:8080 \
  -v ~/.conductor:/root/.conductor \
  -v ~/run-agent:/root/run-agent \
  conductor-loop:dev

# Access at http://localhost:8080
```

### Docker Compose

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  conductor:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ~/.conductor:/root/.conductor
      - ~/run-agent:/root/run-agent
    environment:
      - CONDUCTOR_CONFIG=/root/.conductor/config.yaml
```

**Run:**

```bash
# Start services
docker-compose up

# Run in background
docker-compose up -d

# Stop services
docker-compose down
```

---

## Troubleshooting

### Go Build Errors

**Problem:** `go: module not found`

```bash
# Solution: Download dependencies
go mod download
go mod verify
```

**Problem:** `imports cycle`

```bash
# Solution: Refactor to break circular dependency
# Move shared types to a separate package
```

### Frontend Build Errors

**Problem:** `Module not found: Can't resolve`

```bash
# Solution: Reinstall dependencies
cd frontend
rm -rf node_modules package-lock.json
npm install
```

**Problem:** `npm ERR! peer dependency`

```bash
# Solution: Use --legacy-peer-deps
npm install --legacy-peer-deps
```

### Runtime Errors

**Problem:** `permission denied` when running agent

```bash
# Solution: Make agent CLI executable
chmod +x /path/to/agent-cli
```

**Problem:** `config file not found`

```bash
# Solution: Check config path
ls -la ~/.conductor/config.yaml

# Or specify absolute path
./bin/conductor serve --config /absolute/path/to/config.yaml
```

**Problem:** `agent token is empty`

```bash
# Solution: Set token in config or environment
export AGENT_CLAUDE_TOKEN="your-token"

# Or set in config.yaml
# agents:
#   claude:
#     token: "your-token"
```

### Port Already in Use

**Problem:** `address already in use: :8080`

```bash
# Solution: Kill process using port
lsof -ti:8080 | xargs kill -9

# Or use different port
./bin/conductor serve --config ~/.conductor/config.yaml --port 8081
```

### Test Failures

**Problem:** Tests fail with race condition

```bash
# Solution: Fix race condition
# Use sync.Mutex or channels for shared state

# Identify race with:
go test -race ./...
```

---

## Quick Start Script

**setup-dev.sh:**

```bash
#!/bin/bash
set -e

echo "Setting up conductor-loop development environment..."

# Check prerequisites
command -v go >/dev/null 2>&1 || { echo "Go is not installed"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Node.js is not installed"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "npm is not installed"; exit 1; }

# Install backend dependencies
echo "Installing Go dependencies..."
go mod download

# Install frontend dependencies
echo "Installing npm dependencies..."
cd frontend && npm install && cd ..

# Build binaries
echo "Building binaries..."
go build -o bin/ ./cmd/...

# Create config directory
echo "Creating config directory..."
mkdir -p ~/.conductor

# Create sample config
echo "Creating sample config..."
cat > ~/.conductor/config.yaml <<EOF
agents:
  claude:
    type: claude
    token_file: ~/.claude/token

defaults:
  agent: claude
  timeout: 3600

api:
  host: 0.0.0.0
  port: 8080

storage:
  runs_dir: ~/run-agent
EOF

echo "Setup complete!"
echo "Next steps:"
echo "1. Add your API tokens to ~/.conductor/config.yaml"
echo "2. Run backend: ./bin/conductor serve --config ~/.conductor/config.yaml"
echo "3. Run frontend: cd frontend && npm run dev"
```

**Usage:**

```bash
chmod +x setup-dev.sh
./setup-dev.sh
```

---

## Environment Variables

**Backend:**

```bash
# Config file path
export CONDUCTOR_CONFIG=~/.conductor/config.yaml

# Storage root
export CONDUCTOR_ROOT=~/run-agent

# Disable task execution (API server only)
export CONDUCTOR_DISABLE_TASK_START=true

# Agent tokens
export AGENT_CLAUDE_TOKEN="your-token"
export AGENT_CODEX_TOKEN="your-token"
```

**Frontend:**

```bash
# API base URL (for development)
export VITE_API_BASE_URL=http://localhost:8080

# Enable debug mode
export VITE_DEBUG=true
```

---

## Next Steps

After setting up your development environment:

1. Read the [Contributing Guide](contributing.md)
2. Review the [Architecture Overview](architecture.md)
3. Check the [Testing Guide](testing.md)
4. Pick an issue to work on from GitHub

---

**Last Updated:** 2026-02-05
**Version:** 1.0.0
