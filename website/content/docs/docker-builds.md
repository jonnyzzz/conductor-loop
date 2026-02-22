---
title: "Docker-Only Docs Workflow"
description: "How docs serve/build works without local Hugo installation"
weight: 20
group: "Core Docs"
---

Conductor Loop documentation uses a Hugo container for both local preview and production build.

## Why Docker-Only

- Reproducible Hugo version across contributors and CI
- No local Hugo installation or version drift
- Stable file ownership via UID/GID mapping

## Commands

```bash
# Live preview (port 1313)
./scripts/docs.sh serve

# Production build to website/public
./scripts/docs.sh build

# Output and link checks
./scripts/docs.sh verify
```

## Optional Environment Variables

- `DOCKER_UID` and `DOCKER_GID`: file ownership for generated artifacts
- `HUGO_BASE_URL`: override site base URL for local experiments

Example:

```bash
DOCKER_UID="$(id -u)" DOCKER_GID="$(id -g)" HUGO_BASE_URL="http://localhost:1313/" ./scripts/docs.sh build
```

## CI Usage

CI runs the exact same Docker path via `.github/workflows/docs.yml`.
