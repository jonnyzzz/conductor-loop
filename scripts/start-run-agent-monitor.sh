#!/usr/bin/env bash

set -euo pipefail

script_name="$(basename "$0")"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

usage() {
  cat <<'HELP'
Usage: scripts/start-run-agent-monitor.sh [options] [-- <extra serve args>]

Start run-agent in monitoring-only server mode:
  run-agent serve --disable-task-start

Options:
  --config <path>     Config file path (default: auto-discover)
  --root <path>       Runs root directory (default: $HOME/run-agent)
  --host <host>       API host (default: 127.0.0.1)
  --port <port>       API port (default: 14355)
  --api-key <value>   API key for auth-enabled mode
  --background        Run in background and write pid/log files
  --foreground        Run in foreground (default)
  --pid-file <path>   PID file in background mode
  --log-file <path>   Log file in background mode
  --dry-run           Print resolved command and exit
  -h, --help          Show this help

Environment overrides:
  RUN_AGENT_BIN
  RUN_AGENT_MONITOR_CONFIG, RUN_AGENT_MONITOR_ROOT
  RUN_AGENT_MONITOR_HOST, RUN_AGENT_MONITOR_PORT, RUN_AGENT_MONITOR_API_KEY
  RUN_AGENT_MONITOR_BACKGROUND, RUN_AGENT_MONITOR_DRY_RUN
  RUN_AGENT_MONITOR_STATE_DIR, RUN_AGENT_MONITOR_PID_FILE, RUN_AGENT_MONITOR_LOG_FILE

Compatibility aliases (also accepted):
  CONDUCTOR_CONFIG, CONDUCTOR_ROOT, CONDUCTOR_HOST, CONDUCTOR_PORT, CONDUCTOR_API_KEY
HELP
}

log() {
  printf '[%s] %s\n' "$script_name" "$*"
}

fail() {
  printf '[%s] error: %s\n' "$script_name" "$*" >&2
  exit 1
}

is_true() {
  case "${1:-}" in
    1|true|TRUE|yes|YES|on|ON|y|Y)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

default_config_path() {
  local candidate
  local candidates=()

  candidates+=("$PWD/config.yaml" "$PWD/config.yml" "$PWD/config.hcl")

  if [[ -n "${HOME:-}" ]]; then
    candidates+=(
      "$HOME/.conductor/config.yaml"
      "$HOME/.conductor/config.yml"
      "$HOME/.conductor/config.hcl"
      "$HOME/.config/conductor/config.yaml"
      "$HOME/.config/conductor/config.yml"
      "$HOME/.config/conductor/config.hcl"
    )
  fi

  for candidate in "${candidates[@]}"; do
    if [[ -f "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  return 1
}

resolve_run_agent_binary() {
  local candidate

  if [[ -n "${RUN_AGENT_BIN:-}" ]]; then
    [[ -x "$RUN_AGENT_BIN" ]] || fail "RUN_AGENT_BIN is not executable: $RUN_AGENT_BIN"
    printf '%s\n' "$RUN_AGENT_BIN"
    return 0
  fi

  for candidate in "$repo_root/run-agent" "$repo_root/bin/run-agent"; do
    if [[ -x "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  if candidate="$(command -v run-agent 2>/dev/null)"; then
    printf '%s\n' "$candidate"
    return 0
  fi

  fail "run-agent binary not found. Set RUN_AGENT_BIN or build ./cmd/run-agent"
}

config_path="${RUN_AGENT_MONITOR_CONFIG:-${CONDUCTOR_CONFIG:-}}"
if [[ -z "$config_path" ]]; then
  config_path="$(default_config_path || true)"
fi

root_dir="${RUN_AGENT_MONITOR_ROOT:-${CONDUCTOR_ROOT:-${HOME:-}/run-agent}}"
host="${RUN_AGENT_MONITOR_HOST:-${CONDUCTOR_HOST:-127.0.0.1}}"
port="${RUN_AGENT_MONITOR_PORT:-${CONDUCTOR_PORT:-14355}}"
api_key="${RUN_AGENT_MONITOR_API_KEY:-${CONDUCTOR_API_KEY:-}}"
background="${RUN_AGENT_MONITOR_BACKGROUND:-0}"
dry_run="${RUN_AGENT_MONITOR_DRY_RUN:-0}"
state_dir="${RUN_AGENT_MONITOR_STATE_DIR:-${HOME:-}/.conductor}"
pid_file="${RUN_AGENT_MONITOR_PID_FILE:-$state_dir/run-agent-monitor.pid}"
log_file="${RUN_AGENT_MONITOR_LOG_FILE:-$state_dir/run-agent-monitor.log}"
extra_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)
      [[ $# -ge 2 ]] || fail "missing value for --config"
      config_path="$2"
      shift 2
      ;;
    --root)
      [[ $# -ge 2 ]] || fail "missing value for --root"
      root_dir="$2"
      shift 2
      ;;
    --host)
      [[ $# -ge 2 ]] || fail "missing value for --host"
      host="$2"
      shift 2
      ;;
    --port)
      [[ $# -ge 2 ]] || fail "missing value for --port"
      port="$2"
      shift 2
      ;;
    --api-key)
      [[ $# -ge 2 ]] || fail "missing value for --api-key"
      api_key="$2"
      shift 2
      ;;
    --background)
      background=1
      shift
      ;;
    --foreground)
      background=0
      shift
      ;;
    --pid-file)
      [[ $# -ge 2 ]] || fail "missing value for --pid-file"
      pid_file="$2"
      shift 2
      ;;
    --log-file)
      [[ $# -ge 2 ]] || fail "missing value for --log-file"
      log_file="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --)
      shift
      extra_args=("$@")
      break
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

if [[ -z "$config_path" ]]; then
  fail "config file not found. Pass --config or set RUN_AGENT_MONITOR_CONFIG"
fi
if [[ ! -f "$config_path" ]]; then
  fail "config file does not exist: $config_path"
fi
if [[ ! -r "$config_path" ]]; then
  fail "config file is not readable: $config_path"
fi

if [[ ! "$port" =~ ^[0-9]+$ ]]; then
  fail "port must be numeric: $port"
fi

mkdir -p "$root_dir"
if [[ ! -d "$root_dir" ]]; then
  fail "root path is not a directory: $root_dir"
fi
if [[ ! -w "$root_dir" ]]; then
  fail "root directory is not writable: $root_dir"
fi

binary_path="$(resolve_run_agent_binary)"

cmd=("$binary_path" "serve" "--config" "$config_path" "--root" "$root_dir" "--host" "$host" "--port" "$port" "--disable-task-start")
if [[ -n "$api_key" ]]; then
  cmd+=("--api-key" "$api_key")
fi
if [[ ${#extra_args[@]} -gt 0 ]]; then
  cmd+=("${extra_args[@]}")
fi

url_host="$host"
if [[ "$url_host" == "0.0.0.0" ]]; then
  url_host="127.0.0.1"
fi
base_url="http://${url_host}:${port}"

log "Mode: monitor-only (disable task execution)"
log "Config: $config_path"
log "Root: $root_dir"
log "API URL: $base_url"
log "Web UI: $base_url/ui/"

printf '[%s] Command:' "$script_name"
printf ' %q' "${cmd[@]}"
printf '\n'

if is_true "$dry_run"; then
  exit 0
fi

if is_true "$background"; then
  mkdir -p "$(dirname "$pid_file")"
  mkdir -p "$(dirname "$log_file")"

  if [[ -f "$pid_file" ]]; then
    existing_pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [[ "$existing_pid" =~ ^[0-9]+$ ]] && kill -0 "$existing_pid" 2>/dev/null; then
      fail "already running with pid $existing_pid (pid file: $pid_file)"
    fi
    rm -f "$pid_file"
  fi

  log "Starting in background"
  nohup "${cmd[@]}" >>"$log_file" 2>&1 &
  pid=$!
  printf '%s\n' "$pid" >"$pid_file"

  sleep 0.2
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$pid_file"
    fail "server exited early. Check log: $log_file"
  fi

  log "Started pid=$pid"
  log "PID file: $pid_file"
  log "Log file: $log_file"
  log "Next: tail -f $log_file"
  log "Next: run-agent server watch --server $base_url --project <project-id> --timeout 30m"
  exit 0
fi

log "Starting in foreground (Ctrl+C to stop)"
log "Next: run-agent server status --server $base_url"
exec "${cmd[@]}"
