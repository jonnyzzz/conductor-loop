#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/start-monitor.sh [options] [-- <extra run-agent serve args>]

Start run-agent serve with monitoring-first defaults.

Options:
  --config <path>        Config file path (optional; default: auto-detect if present)
  --root <path>          Runs root directory (default: ./runs)
  --host <host>          Listen host (default: 0.0.0.0)
  --port <port>          Listen port (default: 14355)
  --api-key <key>        Enable API authentication with this key
  --disable-task-start   Disable task execution (default)
  --enable-task-start    Enable task execution (requires config)
  --background           Run in background and write PID/log files
  --foreground           Run in foreground (default)
  --pid-file <path>      PID file for background mode (default: <root>/run-agent-serve.pid)
  --log-file <path>      Log file for background mode (default: <root>/run-agent-serve.log)
  --bin <path>           run-agent binary path override
  --dry-run              Print resolved command and exit
  -h, --help             Show this help

Environment overrides:
  RUN_AGENT_BIN
  RUN_AGENT_CONFIG
  RUN_AGENT_ROOT
  JRUN_RUNS_DIR
  RUN_AGENT_HOST
  RUN_AGENT_PORT
  RUN_AGENT_API_KEY
  RUN_AGENT_DISABLE_TASK_START
  RUN_AGENT_BACKGROUND
  RUN_AGENT_PID_FILE
  RUN_AGENT_LOG_FILE
EOF
}

fail() {
  printf 'start-monitor.sh error: %s\n' "$*" >&2
  exit 1
}

to_lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

is_true() {
  case "$(to_lower "$1")" in
    1|true|yes|y|on)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

abspath_file() {
  local value="$1"
  local dir base
  dir="$(cd "$(dirname "$value")" && pwd)"
  base="$(basename "$value")"
  printf '%s/%s\n' "$dir" "$base"
}

validate_port() {
  local value="$1"
  case "$value" in
    ''|*[!0-9]*)
      fail "port must be a number, got: $value"
      ;;
  esac
  if [ "$value" -lt 1 ] || [ "$value" -gt 65535 ]; then
    fail "port out of range (1..65535): $value"
  fi
}

assert_writable_dir() {
  local path="$1"
  mkdir -p "$path" || fail "cannot create directory: $path"
  local probe="$path/.startup-write-test-$$"
  if ! : >"$probe" 2>/dev/null; then
    fail "directory is not writable: $path"
  fi
  rm -f "$probe"
}

resolve_run_agent_bin() {
  local explicit="$1"
  if [ -n "$explicit" ]; then
    [ -f "$explicit" ] || fail "run-agent binary not found: $explicit"
    [ -x "$explicit" ] || fail "run-agent binary is not executable: $explicit"
    abspath_file "$explicit"
    return 0
  fi

  if command -v run-agent >/dev/null 2>&1; then
    command -v run-agent
    return 0
  fi

  if [ -f "$repo_root/run-agent" ] && [ -x "$repo_root/run-agent" ]; then
    printf '%s\n' "$repo_root/run-agent"
    return 0
  fi

  if [ -f "$repo_root/bin/run-agent" ] && [ -x "$repo_root/bin/run-agent" ]; then
    printf '%s\n' "$repo_root/bin/run-agent"
    return 0
  fi

  fail "run-agent binary not found. Set RUN_AGENT_BIN or build it with: go build -o run-agent ./cmd/run-agent"
}

find_default_config() {
  local home
  home="${HOME:-}"
  local candidates=(
    "$repo_root/config.yaml"
    "$repo_root/config.yml"
    "$repo_root/config.hcl"
  )

  if [ -n "$home" ]; then
    candidates+=(
      "$home/.config/conductor/config.yaml"
      "$home/.config/conductor/config.yml"
      "$home/.config/conductor/config.hcl"
    )
  fi

  local candidate
  for candidate in "${candidates[@]}"; do
    if [ -f "$candidate" ]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  printf '\n'
}

format_http_url() {
  local host="$1"
  local port="$2"
  local url_host="$host"

  if [ "$url_host" = "0.0.0.0" ]; then
    url_host="127.0.0.1"
  fi
  if [ "$url_host" = "::" ]; then
    url_host="::1"
  fi
  case "$url_host" in
    *:*)
      url_host="[$url_host]"
      ;;
  esac
  printf 'http://%s:%s' "$url_host" "$port"
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

config_path="${RUN_AGENT_CONFIG:-${CONDUCTOR_CONFIG:-}}"
root_dir="${RUN_AGENT_ROOT:-${CONDUCTOR_ROOT:-${JRUN_RUNS_DIR:-$repo_root/runs}}}"
host="${RUN_AGENT_HOST:-${CONDUCTOR_HOST:-0.0.0.0}}"
port="${RUN_AGENT_PORT:-${CONDUCTOR_PORT:-14355}}"
api_key="${RUN_AGENT_API_KEY:-${CONDUCTOR_API_KEY:-}}"
disable_task_start="${RUN_AGENT_DISABLE_TASK_START:-1}"
background="${RUN_AGENT_BACKGROUND:-0}"
pid_file="${RUN_AGENT_PID_FILE:-}"
log_file="${RUN_AGENT_LOG_FILE:-}"
run_agent_bin="${RUN_AGENT_BIN:-}"
dry_run=0
extra_args=()

while [ "$#" -gt 0 ]; do
  case "$1" in
    --config)
      [ "$#" -ge 2 ] || fail "missing value for --config"
      config_path="$2"
      shift 2
      ;;
    --root)
      [ "$#" -ge 2 ] || fail "missing value for --root"
      root_dir="$2"
      shift 2
      ;;
    --host)
      [ "$#" -ge 2 ] || fail "missing value for --host"
      host="$2"
      shift 2
      ;;
    --port)
      [ "$#" -ge 2 ] || fail "missing value for --port"
      port="$2"
      shift 2
      ;;
    --api-key)
      [ "$#" -ge 2 ] || fail "missing value for --api-key"
      api_key="$2"
      shift 2
      ;;
    --disable-task-start)
      disable_task_start=1
      shift
      ;;
    --enable-task-start)
      disable_task_start=0
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
      [ "$#" -ge 2 ] || fail "missing value for --pid-file"
      pid_file="$2"
      shift 2
      ;;
    --log-file)
      [ "$#" -ge 2 ] || fail "missing value for --log-file"
      log_file="$2"
      shift 2
      ;;
    --bin)
      [ "$#" -ge 2 ] || fail "missing value for --bin"
      run_agent_bin="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      extra_args=("$@")
      break
      ;;
    *)
      fail "unknown argument: $1 (use --help)"
      ;;
  esac
done

if [ -z "$config_path" ]; then
  config_path="$(find_default_config)"
fi

if [ -n "$config_path" ]; then
  [ -f "$config_path" ] || fail "config file not found: $config_path"
  config_path="$(abspath_file "$config_path")"
fi

validate_port "$port"

if is_true "$disable_task_start"; then
  disable_task_start=1
else
  disable_task_start=0
fi

if is_true "$background"; then
  background=1
else
  background=0
fi

if [ "$disable_task_start" -eq 0 ] && [ -z "$config_path" ]; then
  fail "config is required when task execution is enabled (--enable-task-start)"
fi

assert_writable_dir "$root_dir"
root_dir="$(cd "$root_dir" && pwd)"

if [ -z "$pid_file" ]; then
  pid_file="$root_dir/run-agent-serve.pid"
fi
if [ -z "$log_file" ]; then
  log_file="$root_dir/run-agent-serve.log"
fi
pid_file="$(abspath_file "$pid_file")"
log_file="$(abspath_file "$log_file")"

resolved_bin="$(resolve_run_agent_bin "$run_agent_bin")"

cmd=(
  "$resolved_bin"
  "serve"
  "--root" "$root_dir"
  "--host" "$host"
  "--port" "$port"
)
if [ -n "$config_path" ]; then
  cmd+=("--config" "$config_path")
fi
if [ "$disable_task_start" -eq 1 ]; then
  cmd+=("--disable-task-start")
fi
if [ -n "$api_key" ]; then
  cmd+=("--api-key" "$api_key")
fi
if [ "${#extra_args[@]}" -gt 0 ]; then
  cmd+=("${extra_args[@]}")
fi

base_url="$(format_http_url "$host" "$port")"

printf 'run-agent serve startup configuration:\n'
printf '  Binary: %s\n' "$resolved_bin"
if [ -n "$config_path" ]; then
  printf '  Config: %s\n' "$config_path"
else
  printf '  Config: <none>\n'
fi
printf '  Root:   %s\n' "$root_dir"
printf '  URL:    %s\n' "$base_url"
printf '  UI:     %s/ui/\n' "$base_url"
if [ "$disable_task_start" -eq 1 ]; then
  printf '  Task execution: disabled\n'
else
  printf '  Task execution: enabled\n'
fi
printf '  Command:'
printf ' %q' "${cmd[@]}"
printf '\n'

if [ "$dry_run" -eq 1 ]; then
  printf 'Dry-run mode: command not executed.\n'
  exit 0
fi

if [ "$background" -eq 1 ]; then
  mkdir -p "$(dirname "$pid_file")" "$(dirname "$log_file")"
  if [ -f "$pid_file" ]; then
    existing_pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "$existing_pid" ] && kill -0 "$existing_pid" >/dev/null 2>&1; then
      fail "process already running (pid $existing_pid from $pid_file)"
    fi
    rm -f "$pid_file"
  fi

  nohup "${cmd[@]}" >>"$log_file" 2>&1 &
  child_pid=$!
  printf '%s\n' "$child_pid" >"$pid_file"
  sleep 0.2

  if ! kill -0 "$child_pid" >/dev/null 2>&1; then
    fail "process exited immediately; check logs: $log_file"
  fi

  printf 'Started run-agent serve in background.\n'
  printf '  PID file: %s\n' "$pid_file"
  printf '  Log file: %s\n' "$log_file"
  printf '  Stop: kill "$(cat %q)"\n' "$pid_file"
  exit 0
fi

printf 'Starting run-agent serve in foreground (Ctrl+C to stop)...\n'
exec "${cmd[@]}"
