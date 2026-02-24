#!/usr/bin/env bash

set -euo pipefail

script_name="$(basename "$0")"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

usage() {
  cat <<'HELP'
Usage: scripts/start-conductor.sh [options] [-- <extra server args>]

Start the Conductor API server with practical defaults.
The script prefers a `conductor` binary and falls back to `run-agent serve`.

Options:
  --config <path>     Config file path (default: auto-discover)
  --root <path>       Runs root directory (default: ./runs)
  --host <host>       API host (default: 127.0.0.1)
  --port <port>       API port (default: 14355)
  --api-key <value>   API key for auth-enabled mode
  --disable-task-start
                      Start in monitoring-only mode
  --background        Run in background and write pid/log files
  --foreground        Run in foreground (default)
  --pid-file <path>   PID file in background mode
  --log-file <path>   Log file in background mode
  --bin <path>        Conductor binary path override
  --dry-run           Print resolved command and exit
  -h, --help          Show this help

Environment overrides:
  CONDUCTOR_BIN, RUN_AGENT_BIN
  CONDUCTOR_CONFIG, CONDUCTOR_ROOT, RUNS_DIR, CONDUCTOR_HOST, CONDUCTOR_PORT, CONDUCTOR_API_KEY
  CONDUCTOR_DISABLE_TASK_START
  CONDUCTOR_BACKGROUND, CONDUCTOR_DRY_RUN
  CONDUCTOR_STATE_DIR, CONDUCTOR_PID_FILE, CONDUCTOR_LOG_FILE
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

  candidates+=("$repo_root/config.yaml" "$repo_root/config.yml" "$repo_root/config.hcl")
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

resolve_binary_mode=""
resolve_binary_path=""
resolve_server_binary() {
  local candidate

  if [[ -n "${CONDUCTOR_BIN:-}" ]]; then
    [[ -x "$CONDUCTOR_BIN" ]] || fail "CONDUCTOR_BIN is not executable: $CONDUCTOR_BIN"
    resolve_binary_mode="conductor"
    resolve_binary_path="$CONDUCTOR_BIN"
    return 0
  fi

  if [[ -n "${RUN_AGENT_BIN:-}" ]]; then
    [[ -x "$RUN_AGENT_BIN" ]] || fail "RUN_AGENT_BIN is not executable: $RUN_AGENT_BIN"
    resolve_binary_mode="run-agent"
    resolve_binary_path="$RUN_AGENT_BIN"
    return 0
  fi

  for candidate in "$repo_root/run-agent" "$repo_root/bin/run-agent"; do
    if [[ -x "$candidate" ]]; then
      resolve_binary_mode="run-agent"
      resolve_binary_path="$candidate"
      return 0
    fi
  done

  if candidate="$(command -v run-agent 2>/dev/null)"; then
    resolve_binary_mode="run-agent"
    resolve_binary_path="$candidate"
    return 0
  fi

  fail "no server binary found. Set RUN_AGENT_BIN, or build ./cmd/run-agent"
}

config_path="${CONDUCTOR_CONFIG:-}"
if [[ -z "$config_path" ]]; then
  config_path="$(default_config_path || true)"
fi

root_dir="${CONDUCTOR_ROOT:-${RUNS_DIR:-$repo_root/runs}}"
host="${CONDUCTOR_HOST:-127.0.0.1}"
port="${CONDUCTOR_PORT:-14355}"
api_key="${CONDUCTOR_API_KEY:-}"
background="${CONDUCTOR_BACKGROUND:-0}"
dry_run="${CONDUCTOR_DRY_RUN:-0}"
disable_task_start="${CONDUCTOR_DISABLE_TASK_START:-0}"
state_dir="${CONDUCTOR_STATE_DIR:-${HOME:-}/.conductor}"
pid_file="${CONDUCTOR_PID_FILE:-$state_dir/conductor.pid}"
log_file="${CONDUCTOR_LOG_FILE:-$state_dir/conductor.log}"
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
    --disable-task-start)
      disable_task_start=1
      shift
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
    --bin)
      [[ $# -ge 2 ]] || fail "missing value for --bin"
      CONDUCTOR_BIN="$2"
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
  fail "config file not found. Pass --config or set CONDUCTOR_CONFIG"
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

resolve_server_binary

cmd=("$resolve_binary_path")
if [[ "$resolve_binary_mode" == "run-agent" ]]; then
  cmd+=("serve")
fi
cmd+=("--config" "$config_path" "--root" "$root_dir" "--host" "$host" "--port" "$port")
if [[ -n "$api_key" ]]; then
  cmd+=("--api-key" "$api_key")
fi
if is_true "$disable_task_start"; then
  cmd+=("--disable-task-start")
fi
if [[ ${#extra_args[@]} -gt 0 ]]; then
  cmd+=("${extra_args[@]}")
fi

url_host="$host"
if [[ "$url_host" == "0.0.0.0" ]]; then
  url_host="127.0.0.1"
fi
base_url="http://${url_host}:${port}"

log "Backend: $resolve_binary_mode"
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
  log "Next: run-agent server status --server $base_url"
  exit 0
fi

log "Starting in foreground (Ctrl+C to stop)"
log "Next: open $base_url/ui/"
exec "${cmd[@]}"
