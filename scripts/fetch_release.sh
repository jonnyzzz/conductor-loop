#!/usr/bin/env bash
# fetch_release.sh — Download a conductor-loop release binary from GitHub.
#
# STATUS: STUB — no GitHub releases exist yet. This script documents the
# intended download pattern. Use deploy_locally.sh for local dev builds.
#
# Intended Usage (once releases are published):
#   ./scripts/fetch_release.sh [--version <tag>]
#
# Options:
#   --version VERSION   Specific release tag (e.g. v0.76.1). Default: latest.
#   --help              Show this message
#
# The script will:
#   1. Fetch the GitHub releases JSON from the API.
#   2. Select the appropriate asset for the current OS/arch.
#   3. Download and verify SHA256 checksum.
#   4. Install to ~/conductor-loop/binaries/<version>/run-agent.
#   5. Update ~/conductor-loop/binaries/_latest/ symlink.
#
# Pattern reference: ~/Work/devrig/cli/bootstrap/devrig (devrig bootstrap script).
# That script reads a config YAML with URL+SHA512, downloads, verifies, and caches.
# This script will use the GitHub Releases API instead.
#
# GitHub Releases API endpoint:
#   Latest:   https://api.github.com/repos/jonnyzzz/conductor-loop/releases/latest
#   Specific: https://api.github.com/repos/jonnyzzz/conductor-loop/releases/tags/<tag>
#
# Asset naming convention (to be established with release workflow):
#   run-agent-<os>-<arch>         (Linux/macOS)
#   run-agent-<os>-<arch>.exe     (Windows)
#   SHA256SUMS                    (checksum file)

set -euo pipefail

GITHUB_REPO="jonnyzzz/conductor-loop"
BINARIES_HOME="${HOME}/conductor-loop/binaries"

fail() {
  printf 'fetch_release.sh: %s\n' "$*" >&2
  exit 1
}

fail "fetch_release.sh: No GitHub releases available yet. Use scripts/deploy_locally.sh for local development builds."

# ============================================================
# FUTURE IMPLEMENTATION (uncomment once releases are published)
# ============================================================
#
# detect_os() {
#   case "$(uname -s)" in
#     Linux) printf 'linux' ;;
#     Darwin) printf 'darwin' ;;
#     *) fail "unsupported OS: $(uname -s)" ;;
#   esac
# }
#
# detect_arch() {
#   case "$(uname -m)" in
#     x86_64|amd64) printf 'amd64' ;;
#     arm64|aarch64) printf 'arm64' ;;
#     *) fail "unsupported arch: $(uname -m)" ;;
#   esac
# }
#
# VERSION=""
# while [[ $# -gt 0 ]]; do
#   case "$1" in
#     --version) VERSION="$2"; shift 2 ;;
#     --help|-h) grep '^#' "$0" | sed 's/^# \?//'; exit 0 ;;
#     *) fail "unknown argument: $1" ;;
#   esac
# done
#
# OS="$(detect_os)"
# ARCH="$(detect_arch)"
# ASSET_NAME="run-agent-${OS}-${ARCH}"
#
# if [[ -z "$VERSION" ]]; then
#   API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
# else
#   API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${VERSION}"
# fi
#
# echo "==> Fetching release metadata: ${API_URL}"
# RELEASE_JSON="$(curl -fsSL "$API_URL")"
# VERSION="$(printf '%s' "$RELEASE_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['tag_name'])")"
#
# DOWNLOAD_URL="$(printf '%s' "$RELEASE_JSON" | python3 -c "
# import sys, json
# assets = json.load(sys.stdin)['assets']
# name = '${ASSET_NAME}'
# match = next((a['browser_download_url'] for a in assets if a['name'] == name), None)
# if not match: sys.exit(1)
# print(match)
# ")"
#
# VERSIONED_DIR="${BINARIES_HOME}/${VERSION}"
# BINARY_PATH="${VERSIONED_DIR}/run-agent"
# mkdir -p "$VERSIONED_DIR"
#
# echo "==> Downloading ${DOWNLOAD_URL}"
# curl -fsSL -o "$BINARY_PATH" "$DOWNLOAD_URL"
# chmod +x "$BINARY_PATH"
#
# # Verify SHA256 checksum.
# CHECKSUM_URL="${DOWNLOAD_URL%/*}/SHA256SUMS"
# EXPECTED="$(curl -fsSL "$CHECKSUM_URL" | grep " ${ASSET_NAME}$" | awk '{print $1}')"
# ACTUAL="$(sha256sum "$BINARY_PATH" | awk '{print $1}')"
# if [[ "$EXPECTED" != "$ACTUAL" ]]; then
#   rm -f "$BINARY_PATH"
#   fail "SHA256 mismatch: expected ${EXPECTED}, got ${ACTUAL}"
# fi
#
# # Update _latest.
# LATEST_DIR="${BINARIES_HOME}/_latest"
# rm -rf "$LATEST_DIR"
# mkdir -p "$LATEST_DIR"
# ln -sf "$BINARY_PATH" "${LATEST_DIR}/run-agent"
#
# echo "==> Installed ${VERSION} to ${BINARY_PATH}"
# echo "==> _latest -> ${VERSIONED_DIR}"
