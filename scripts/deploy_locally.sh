#!/usr/bin/env bash
# deploy_locally.sh — Build run-agent from source and deploy to ~/conductor-loop/binaries/.
#
# Usage:
#   ./scripts/deploy_locally.sh [--version <version>]
#
# Options:
#   --version VERSION   Override version string (default: git describe or dev-<hash>)
#   --help              Show this message
#
# After running, ~/conductor-loop/binaries/_latest/run-agent will point to the
# newly built binary. run-agent.cmd will find it automatically.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARIES_HOME="${HOME}/conductor-loop/binaries"

fail() {
  printf 'deploy_locally.sh: %s\n' "$*" >&2
  exit 1
}

usage() {
  grep '^#' "$0" | sed 's/^# \?//'
  exit 0
}

detect_version() {
  local v
  if v="$(git -C "$REPO_DIR" describe --tags --exact-match 2>/dev/null)"; then
    printf '%s' "$v"
  elif v="$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null)"; then
    printf 'dev-%s' "$v"
  else
    printf 'dev-unknown'
  fi
}

VERSION=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --help|-h) usage ;;
    *) fail "unknown argument: $1" ;;
  esac
done

if [[ -z "$VERSION" ]]; then
  VERSION="$(detect_version)"
fi

echo "==> Building run-agent ${VERSION}"

VERSIONED_DIR="${BINARIES_HOME}/${VERSION}"
BINARY_PATH="${VERSIONED_DIR}/run-agent"
LATEST_DIR="${BINARIES_HOME}/_latest"

mkdir -p "$VERSIONED_DIR"

# Build with version injected via ldflags.
(
  cd "$REPO_DIR"
  CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "$BINARY_PATH" \
    ./cmd/run-agent
)

chmod +x "$BINARY_PATH"
echo "==> Installed: ${BINARY_PATH}"

# Update _latest to point to this version's directory.
rm -rf "$LATEST_DIR"
mkdir -p "$LATEST_DIR"
ln -sf "$BINARY_PATH" "${LATEST_DIR}/run-agent"
echo "==> Updated _latest -> ${VERSIONED_DIR}"

# Print version for verification.
echo "==> Version check:"
"${LATEST_DIR}/run-agent" --version 2>/dev/null || "${LATEST_DIR}/run-agent" version 2>/dev/null || true

echo ""
echo "Deploy complete. Binary at:"
echo "  ${BINARY_PATH}"
echo ""
echo "Latest symlink:"
echo "  ${LATEST_DIR}/run-agent -> ${BINARY_PATH}"
