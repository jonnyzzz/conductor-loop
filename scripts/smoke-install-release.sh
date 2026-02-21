#!/usr/bin/env bash

set -euo pipefail

log() {
  printf '[installer-smoke] %s\n' "$*"
}

fail() {
  printf '[installer-smoke] error: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: scripts/smoke-install-release.sh [options]

Smoke-test install.sh against release-like artifacts on the current platform.

Options:
  --dist-dir <path>       Directory with release artifacts (default: ./dist)
  --install-script <path> Path to install.sh (default: ./install.sh)
  --asset-name <name>     Asset name override (default: run-agent-<os>-<arch>)
  --keep-temp             Keep temporary directory for debugging
  -h, --help              Show this help
EOF
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

detect_os() {
  case "$(uname -s)" in
    Linux)
      printf 'linux\n'
      ;;
    Darwin)
      printf 'darwin\n'
      ;;
    *)
      fail "unsupported operating system $(uname -s); expected Linux or Darwin"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64\n'
      ;;
    arm64|aarch64)
      printf 'arm64\n'
      ;;
    *)
      fail "unsupported architecture $(uname -m); expected amd64 or arm64"
      ;;
  esac
}

sha256_file() {
  file_path="$1"
  if has_cmd sha256sum; then
    sha256sum "$file_path" | awk '{print $1}'
    return
  fi
  if has_cmd shasum; then
    shasum -a 256 "$file_path" | awk '{print $1}'
    return
  fi
  fail 'neither sha256sum nor shasum is available'
}

wait_for_file() {
  file_path="$1"
  attempts="$2"
  while [ "$attempts" -gt 0 ]; do
    if [ -s "$file_path" ]; then
      return 0
    fi
    attempts=$((attempts - 1))
    sleep 0.1
  done
  return 1
}

run_installer() {
  mirror_base="$1"
  fallback_base="$2"

  RUN_AGENT_INSTALL_DIR="$install_dir" \
  RUN_AGENT_DOWNLOAD_BASE="$mirror_base" \
  RUN_AGENT_FALLBACK_DOWNLOAD_BASE="$fallback_base" \
    bash "$install_script"
}

assert_installed_hash() {
  expected_asset="$1"
  expected_hash="$(sha256_file "$expected_asset")"
  actual_hash="$(sha256_file "$installed_binary")"
  [ "$actual_hash" = "$expected_hash" ] || fail "installed binary hash mismatch: expected ${expected_hash}, got ${actual_hash}"
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
dist_dir="${DIST_DIR:-$repo_root/dist}"
install_script="${INSTALL_SCRIPT:-$repo_root/install.sh}"
asset_name=""
keep_temp=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dist-dir)
      [ "$#" -ge 2 ] || fail 'missing value for --dist-dir'
      dist_dir="$2"
      shift 2
      ;;
    --install-script)
      [ "$#" -ge 2 ] || fail 'missing value for --install-script'
      install_script="$2"
      shift 2
      ;;
    --asset-name)
      [ "$#" -ge 2 ] || fail 'missing value for --asset-name'
      asset_name="$2"
      shift 2
      ;;
    --keep-temp)
      keep_temp=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

has_cmd python3 || fail 'python3 is required to run local release-like HTTP server'
[ -f "$install_script" ] || fail "install script not found: $install_script"

if [ -z "$asset_name" ]; then
  goos="$(detect_os)"
  goarch="$(detect_arch)"
  asset_name="run-agent-${goos}-${goarch}"
fi

source_asset="${dist_dir}/${asset_name}"
[ -f "$source_asset" ] || fail "required release artifact not found: $source_asset"

tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t installer-smoke)"
server_pid=""
port_file="$tmp_dir/server.port"

cleanup() {
  if [ -n "$server_pid" ] && kill -0 "$server_pid" >/dev/null 2>&1; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
  if [ "$keep_temp" -ne 1 ]; then
    rm -rf "$tmp_dir"
  else
    log "kept temp directory: $tmp_dir"
  fi
}
trap cleanup EXIT INT TERM

mirror_latest_dir="$tmp_dir/http-root/mirror/releases/latest/download"
fallback_latest_dir="$tmp_dir/http-root/github/releases/latest/download"
mkdir -p "$mirror_latest_dir" "$fallback_latest_dir"

asset_v1="$tmp_dir/${asset_name}.v1"
asset_v2="$tmp_dir/${asset_name}.v2"
cp "$source_asset" "$asset_v1"
cp "$source_asset" "$asset_v2"
chmod 0755 "$asset_v1" "$asset_v2"

# Keep a valid executable while making update artifacts byte-distinct.
printf '\ninstaller-smoke-update-marker\n' >>"$asset_v2"

mirror_latest_asset="$mirror_latest_dir/$asset_name"
fallback_latest_asset="$fallback_latest_dir/$asset_name"
cp "$asset_v1" "$mirror_latest_asset"
cp "$asset_v1" "$fallback_latest_asset"

python3 - "$tmp_dir/http-root" "$port_file" <<'PY' >/dev/null 2>&1 &
import http.server
import pathlib
import socketserver
import sys

root_dir = sys.argv[1]
port_file = pathlib.Path(sys.argv[2])

class Handler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, fmt, *args):
        return

def handler(*args, **kwargs):
    return Handler(*args, directory=root_dir, **kwargs)

with socketserver.TCPServer(("127.0.0.1", 0), handler) as httpd:
    port_file.write_text(str(httpd.server_address[1]))
    httpd.serve_forever()
PY
server_pid="$!"

wait_for_file "$port_file" 80 || fail 'timed out waiting for local HTTP server port'
if ! kill -0 "$server_pid" >/dev/null 2>&1; then
  fail 'local HTTP server failed to start'
fi

port="$(cat "$port_file")"
base_url="http://127.0.0.1:${port}"
mirror_base_releases="${base_url}/mirror/releases"
mirror_base_download="${base_url}/mirror/releases/download"
mirror_base_latest="${base_url}/mirror/releases/latest/download"
fallback_base_latest="${base_url}/github/releases/latest/download"

install_dir="$tmp_dir/install/bin"
mkdir -p "$install_dir"
installed_binary="$install_dir/run-agent"

log "step 1/3: install from mirror using /releases base (normalizes to latest/download)"
run_installer "$mirror_base_releases" "$fallback_base_latest"
test -x "$installed_binary" || fail "installed binary is missing or not executable: $installed_binary"
assert_installed_hash "$asset_v1"
hash_step1="$(sha256_file "$installed_binary")"

log "step 2/3: update from mirror using /releases/download base"
cp "$asset_v2" "$mirror_latest_asset"
run_installer "$mirror_base_download" "$fallback_base_latest"
assert_installed_hash "$asset_v2"
hash_step2="$(sha256_file "$installed_binary")"
[ "$hash_step1" != "$hash_step2" ] || fail 'update flow did not change installed binary'

log "step 3/3: fallback to secondary base when mirror latest asset is missing"
rm -f "$mirror_latest_asset"
cp "$asset_v1" "$fallback_latest_asset"
run_installer "$mirror_base_latest" "$fallback_base_latest"
assert_installed_hash "$asset_v1"

version_output="$("$installed_binary" --version 2>&1 || true)"
if [ -z "$version_output" ]; then
  fail 'installed binary did not return output for --version'
fi

log "PASS: install/update/latest/fallback smoke checks succeeded for asset ${asset_name}"
