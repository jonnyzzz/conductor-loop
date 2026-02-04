# Task: Set Up Tooling and CI/CD

**Task ID**: bootstrap-03
**Phase**: Bootstrap
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Set up development tooling, Docker, and CI/CD pipelines.

## Required Actions

1. **Docker Setup**
   Create Dockerfile:
   - Multi-stage build (Go builder + minimal runtime)
   - Copy conductor binary
   - Expose API port (default: 8080)
   - Volume for /data (run storage)

2. **Docker Compose**
   Services:
   - conductor: Main service
   - postgres: (optional, for future metadata storage)
   - nginx: (optional, for frontend)

3. **GitHub Actions Workflows**
   Create .github/workflows/:
   - test.yml: Run tests on push/PR
   - build.yml: Build binaries on release
   - docker.yml: Build and push Docker image
   - lint.yml: Run golangci-lint

4. **Monitoring Scripts**
   Create:
   - watch-agents.sh (60s polling)
   - monitor-agents.py (live console monitor)

## Success Criteria
- Docker builds successfully
- docker-compose up works
- GitHub Actions validate

## References
- THE_PROMPT_v5.md: Agent Execution and Traceability

## Output
Log to MESSAGE-BUS.md:
- FACT: Docker image builds
- FACT: CI/CD pipelines configured
