---
title: "Getting Started"
description: "Run the docs site and core workflows quickly"
weight: 10
group: "Core Docs"
---

## What You Need

- Docker 20.10+ with Docker Compose V2 (`docker compose`)
- Git checkout of `conductor-loop`

No local Hugo binary is required.

## Preview This Documentation Site

```bash
./scripts/docs.sh serve
```

Open `http://localhost:1313/`.

## Build The Static Site

```bash
./scripts/docs.sh build
```

Generated files are written to `website/public/`.

## Validate Generated Artifacts And Internal Links

```bash
./scripts/docs.sh verify
```

## Next Reads

- [Docker docs workflow]({{< relref "/docs/docker-builds.md" >}})
- [System architecture]({{< relref "/docs/architecture.md" >}})
- [Message bus protocol]({{< relref "/docs/message-bus.md" >}})
