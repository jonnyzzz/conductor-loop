#!/usr/bin/env bash
# scripts/bootstrap.sh — Unified install/update script for run-agent
#
# Downloads the latest (or specified) run-agent binary from GitHub Releases,
# verifies SHA256, and installs it to ~/.local/bin/run-agent.
#
# Usage:
#   ./scripts/bootstrap.sh [--force] [--version vX.Y.Z] [--install-dir DIR]
#
# Options:
#   --force             Reinstall even if already at the latest version
#   --version TAG       Install a specific release tag (default: latest)
#   --install-dir DIR   Install destination directory (default: ~/.local/bin)
#   --dry-run           Show what would be downloaded without installing
#   -h, --help          Show this help

set -euo pipefail

# ── constants ────────────────────────────────────────────────────────────────
REPO="jonnyzzz/conductor-loop"
BINARY_NAME="run-agent"
DOWNLOAD_BASE="https://github.com/${REPO}/releases/download"
API_LATEST="https://api.github.com/repos/${REPO}/releases/latest"
FALLBACK_DOMAIN="run-agent.jonnyzzz.com"

# ── logging ──────────────────────────────────────────────────────────────────
log()  { printf '[bootstrap] %s\n' "$*"; }
warn() { printf '[bootstrap] warning: %s\n' "$*" >&2; }
fail() { printf '[bootstrap] error: %s\n' "$*" >&2; exit 1; }

# ── helpers ──────────────────────────────────────────────────────────────────
has_cmd() { command -v "$1" >/dev/null 2>&1; }

download_file() {
  local url="$1" out="$2"
  if has_cmd curl; then
    curl -fL --retry 3 --retry-delay 1 --connect-timeout 15 -o "$out" "$url" \
      || fail "curl failed: $url"
  elif has_cmd wget; then
    wget -q --tries=3 -O "$out" "$url" \
      || fail "wget failed: $url"
  else
    fail "neither curl nor wget found; install one and re-run"
  fi
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$os" in
    linux)  os="linux"  ;;
    darwin) os="darwin" ;;
    *)      fail "unsupported OS: $os (use WSL2 on Windows)" ;;
  esac

  case "$arch" in
    x86_64|amd64)   arch="amd64"  ;;
    arm64|aarch64)  arch="arm64"  ;;
    *)              fail "unsupported architecture: $arch" ;;
  esac

  echo "${os}-${arch}"
}

fetch_latest_version() {
  local tmp
  tmp="$(mktemp)"
  if download_file "$API_LATEST" "$tmp" 2>/dev/null; then
    # Try to parse with python3, then python, then grep
    local ver
    if has_cmd python3; then
      ver="$(python3 -c "import json,sys; d=json.load(open('$tmp')); print(d['tag_name'])" 2>/dev/null || true)"
    elif has_cmd python; then
      ver="$(python -c "import json,sys; d=json.load(open('$tmp')); print(d['tag_name'])" 2>/dev/null || true)"
    fi
    if [[ -z "${ver:-}" ]]; then
      ver="$(grep -o '"tag_name": *"[^"]*"' "$tmp" | head -1 | sed 's/.*: *"//;s/"//')"
    fi
    rm -f "$tmp"
    [[ -n "$ver" ]] || fail "could not parse latest version from GitHub API"
    echo "$ver"
  else
    rm -f "$tmp"
    fail "could not fetch latest release from GitHub API"
  fi
}

current_version() {
  local bin
  bin="$(command -v "$BINARY_NAME" 2>/dev/null || echo "")"
  if [[ -z "$bin" ]]; then
    echo "not-installed"
    return
  fi
  "$bin" version 2>/dev/null || echo "unknown"
}

verify_sha256() {
  local file="$1" expected_file="$2"
  local expected actual
  expected="$(awk '{print $1}' "$expected_file")"
  if has_cmd sha256sum; then
    actual="$(sha256sum "$file" | awk '{print $1}')"
  elif has_cmd shasum; then
    actual="$(shasum -a 256 "$file" | awk '{print $1}')"
  else
    warn "no sha256sum or shasum found — skipping checksum verification"
    return 0
  fi
  [[ "$actual" == "$expected" ]] \
    || fail "SHA256 mismatch! expected=$expected actual=$actual — aborting"
}

# ── argument parsing ──────────────────────────────────────────────────────────
force=0
dry_run=0
target_version=""
install_dir="${HOME}/.local/bin"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --force)          force=1; shift ;;
    --dry-run)        dry_run=1; shift ;;
    --version)        target_version="$2"; shift 2 ;;
    --install-dir)    install_dir="$2"; shift 2 ;;
    -h|--help)
      head -20 "$0" | grep '^#' | sed 's/^# \?//'
      exit 0
      ;;
    *)
      fail "unknown argument: $1 (use --help)"
      ;;
  esac
done

# ── main ──────────────────────────────────────────────────────────────────────
platform="$(detect_platform)"
log "platform: $platform"

# Resolve target version
if [[ -z "$target_version" ]]; then
  log "fetching latest release version..."
  target_version="$(fetch_latest_version)"
fi
log "target version: $target_version"

# Check current version
cur="$(current_version)"
log "current version: $cur"

if [[ "$force" -eq 0 && "$cur" == "$target_version" ]]; then
  log "Already up to date ($target_version). Use --force to reinstall."
  exit 0
fi

# Build download URL
bin_file="${BINARY_NAME}-${platform}"
sha_file="${bin_file}.sha256"
download_url="${DOWNLOAD_BASE}/${target_version}/${bin_file}"
sha_url="${DOWNLOAD_BASE}/${target_version}/${sha_file}"

if [[ "$dry_run" -eq 1 ]]; then
  log "dry-run: would download $download_url"
  log "dry-run: would verify $sha_url"
  log "dry-run: would install to ${install_dir}/${BINARY_NAME}"
  exit 0
fi

# Download to temp dir
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

tmp_bin="${tmp_dir}/${bin_file}"
tmp_sha="${tmp_dir}/${sha_file}"

log "downloading ${BINARY_NAME} ${target_version}..."
download_file "$download_url" "$tmp_bin"

log "downloading checksum..."
download_file "$sha_url" "$tmp_sha"

log "verifying SHA256..."
verify_sha256 "$tmp_bin" "$tmp_sha"
log "checksum OK"

# Install
mkdir -p "$install_dir"
chmod +x "$tmp_bin"
# Atomic move
mv "$tmp_bin" "${install_dir}/${BINARY_NAME}"
log "installed to ${install_dir}/${BINARY_NAME}"

# Verify
installed_ver="$("${install_dir}/${BINARY_NAME}" version 2>/dev/null || echo "unknown")"
log "verified: ${BINARY_NAME} version = $installed_ver"

if [[ "$installed_ver" == "unknown" ]]; then
  warn "binary installed but 'run-agent version' returned unknown"
else
  log "SUCCESS: run-agent $installed_ver installed"
fi

# PATH hint
if ! echo "$PATH" | grep -q "$install_dir"; then
  warn "NOTE: ${install_dir} is not in your PATH."
  warn "Add this to your shell profile:"
  warn "  export PATH=\"\$PATH:${install_dir}\""
fi
