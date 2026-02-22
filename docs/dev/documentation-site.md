# Documentation Website

Conductor Loop documentation website is built with Hugo in Docker only.

## Why Docker-Only

- Contributors and CI use the same Hugo image/version.
- No local Hugo installation or local version drift.
- Generated files keep expected ownership through UID/GID mapping.

## Commands

From repository root:

```bash
./scripts/docs.sh serve
```

Runs a local docs server at `http://localhost:1313/`.

```bash
./scripts/docs.sh build
```

Builds static files into `website/public/`.

```bash
./scripts/docs.sh verify
```

Checks expected generated pages and key internal links.

## Environment Overrides

- `DOCKER_UID` and `DOCKER_GID`: file ownership for generated output.
- `HUGO_BASE_URL`: override base URL passed into Hugo.

Example:

```bash
DOCKER_UID="$(id -u)" DOCKER_GID="$(id -g)" HUGO_BASE_URL="http://localhost:1313/" ./scripts/docs.sh build
```

## CI

Workflow `.github/workflows/docs.yml` runs the same Docker-only build path.
