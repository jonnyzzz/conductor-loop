#!/usr/bin/env bash

set -euo pipefail

log() {
  printf '[run-agent installer] %s\n' "$*"
}

warn() {
  printf '[run-agent installer] warning: %s\n' "$*" >&2
}

fail() {
  printf '[run-agent installer] error: %s\n' "$*" >&2
  exit 1
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

download_file() {
  url="$1"
  out_file="$2"

  if has_cmd curl; then
    curl -fL --retry 3 --retry-delay 1 --connect-timeout 15 -o "$out_file" "$url"
    return
  fi

  if has_cmd wget; then
    wget --tries=3 --timeout=15 -O "$out_file" "$url"
    return
  fi

  fail 'neither curl nor wget is available; install one of them and retry'
}

detect_os() {
  uname_s="$(uname -s)"
  case "$uname_s" in
    Linux)
      echo linux
      ;;
    Darwin)
      echo darwin
      ;;
    *)
      fail "unsupported operating system: ${uname_s}; supported: Linux and macOS"
      ;;
  esac
}

detect_arch() {
  uname_m="$(uname -m)"
  case "$uname_m" in
    x86_64|amd64)
      echo amd64
      ;;
    arm64|aarch64)
      echo arm64
      ;;
    *)
      fail "unsupported architecture: ${uname_m}; supported: amd64 and arm64"
      ;;
  esac
}

normalize_download_base() {
  base="${1%/}"

  case "$base" in
    */releases/latest/download)
      printf '%s\n' "$base"
      ;;
    */releases/download)
      printf '%s/latest/download\n' "${base%/download}"
      ;;
    */releases)
      printf '%s/latest/download\n' "$base"
      ;;
    *)
      printf '%s\n' "$base"
      ;;
  esac
}

install_binary() {
  src_file="$1"
  install_path="$2"

  install_dir="$(dirname "$install_path")"
  if ! mkdir -p "$install_dir" 2>/dev/null; then
    if has_cmd sudo; then
      log "Using sudo to create ${install_dir}"
      sudo mkdir -p "$install_dir"
    else
      fail "failed to create ${install_dir}; check permissions or set RUN_AGENT_INSTALL_DIR"
    fi
  fi

  if has_cmd install; then
    if install -m 0755 "$src_file" "$install_path"; then
      return
    fi
    if has_cmd sudo; then
      log "Using sudo to install into ${install_path}"
      if sudo install -m 0755 "$src_file" "$install_path"; then
        return
      fi
    fi
    fail "failed to install to ${install_path}; check permissions or set RUN_AGENT_INSTALL_DIR"
  fi

  if cp "$src_file" "$install_path" 2>/dev/null; then
    chmod 0755 "$install_path"
    return
  fi
  if has_cmd sudo; then
    log "Using sudo to copy into ${install_path}"
    if sudo cp "$src_file" "$install_path" && sudo chmod 0755 "$install_path"; then
      return
    fi
  fi
  fail "failed to copy to ${install_path}; check permissions or set RUN_AGENT_INSTALL_DIR"
}

main() {
  mirror_download_base="$(normalize_download_base "${RUN_AGENT_DOWNLOAD_BASE:-https://run-agent.jonnyzzz.com/releases/latest/download}")"
  fallback_download_base="$(normalize_download_base "${RUN_AGENT_FALLBACK_DOWNLOAD_BASE:-https://github.com/jonnyzzz/conductor-loop/releases/latest/download}")"
  install_dir="${RUN_AGENT_INSTALL_DIR:-/usr/local/bin}"

  goos="$(detect_os)"
  goarch="$(detect_arch)"

  asset_name="run-agent-${goos}-${goarch}"
  install_path="${install_dir}/run-agent"

  tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t run-agent-installer)"
  tmp_asset="${tmp_dir}/${asset_name}"
  trap 'rm -rf "${tmp_dir}"' EXIT INT TERM

  mirror_url="${mirror_download_base}/${asset_name}"
  fallback_url="${fallback_download_base}/${asset_name}"

  downloaded=0
  log "Downloading latest ${asset_name} from mirror: ${mirror_url}"
  if download_file "$mirror_url" "$tmp_asset"; then
    if [ -s "$tmp_asset" ]; then
      downloaded=1
    else
      warn "mirror download produced an empty file"
    fi
  else
    warn 'mirror download failed'
  fi

  if [ "$downloaded" -ne 1 ]; then
    log "Falling back to secondary release asset URL: ${fallback_url}"
    download_file "$fallback_url" "$tmp_asset" || fail "failed to download release asset: ${asset_name}"
  fi

  [ -s "$tmp_asset" ] || fail "downloaded file is empty: ${tmp_asset}"

  install_binary "$tmp_asset" "$install_path"
  log "Installed run-agent to ${install_path}"

  if version_output="$($install_path --version 2>&1)"; then
    printf '%s\n' "$version_output"
  else
    warn 'installed binary, but failed to execute run-agent --version'
  fi
}

main "$@"
