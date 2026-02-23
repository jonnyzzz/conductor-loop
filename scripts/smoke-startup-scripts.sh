#!/usr/bin/env bash

set -euo pipefail

log() {
  printf '[startup-smoke] %s\n' "$*"
}

fail() {
  printf '[startup-smoke] error: %s\n' "$*" >&2
  exit 1
}

wait_for_file() {
  local file_path="$1"
  local attempts="$2"
  while [ "$attempts" -gt 0 ]; do
    if [ -f "$file_path" ]; then
      return 0
    fi
    attempts=$((attempts - 1))
    sleep 0.1
  done
  return 1
}

usage() {
  cat <<'EOF'
Usage: scripts/smoke-startup-scripts.sh [options]

Smoke-test startup wrappers with fake binaries.

Options:
  --keep-temp   Keep temporary working directory
  -h, --help    Show this help
EOF
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
keep_temp=0

while [ "$#" -gt 0 ]; do
  case "$1" in
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

start_conductor="$repo_root/scripts/start-conductor.sh"
start_monitor="$repo_root/scripts/start-monitor.sh"

[ -x "$start_conductor" ] || fail "missing executable script: $start_conductor"
[ -x "$start_monitor" ] || fail "missing executable script: $start_monitor"

tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t startup-smoke)"
pids_to_kill=()

cleanup() {
  local pid
  if [ "${#pids_to_kill[@]}" -gt 0 ]; then
    for pid in "${pids_to_kill[@]}"; do
      if kill -0 "$pid" >/dev/null 2>&1; then
        kill "$pid" >/dev/null 2>&1 || true
      fi
    done
  fi
  if [ "$keep_temp" -ne 1 ]; then
    rm -rf "$tmp_dir"
  else
    log "kept temp directory: $tmp_dir"
  fi
}
trap cleanup EXIT INT TERM

bin_dir="$tmp_dir/bin"
mkdir -p "$bin_dir"

cat >"$bin_dir/fake-conductor" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
args_file="${SMOKE_ARGS_FILE:?missing SMOKE_ARGS_FILE}"
{
  printf 'argv0=%s\n' "$0"
  for arg in "$@"; do
    printf '%s\n' "$arg"
  done
} >"$args_file"
sleep "${SMOKE_FAKE_SLEEP:-30}"
EOF
chmod +x "$bin_dir/fake-conductor"

cat >"$bin_dir/fake-run-agent" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
args_file="${SMOKE_ARGS_FILE:?missing SMOKE_ARGS_FILE}"
{
  printf 'argv0=%s\n' "$0"
  for arg in "$@"; do
    printf '%s\n' "$arg"
  done
} >"$args_file"
sleep "${SMOKE_FAKE_SLEEP:-30}"
EOF
chmod +x "$bin_dir/fake-run-agent"

config_file="$tmp_dir/config.yaml"
cat >"$config_file" <<'EOF'
agents:
  codex:
    type: codex
    token: fake
defaults:
  agent: codex
storage:
  runs_dir: ./runs
EOF

runs_root="$tmp_dir/runs-root"
mkdir -p "$runs_root"

log "1/4 validating dry-run output for conductor startup wrapper"
conductor_dry="$tmp_dir/conductor-dry.txt"
CONDUCTOR_BIN="$bin_dir/fake-conductor" \
  "$start_conductor" \
  --dry-run \
  --config "$config_file" \
  --root "$runs_root" \
  --host 127.0.0.1 \
  --port 19081 >"$conductor_dry"
rg -q '\[start-conductor.sh\] Backend:' "$conductor_dry" || fail "missing conductor backend line in dry-run output"
rg -q 'Web UI: http://127.0.0.1:19081/ui/' "$conductor_dry" || fail "missing conductor UI URL in dry-run output"

log "2/4 validating background mode for conductor startup wrapper"
conductor_pid_file="$tmp_dir/conductor.pid"
conductor_log_file="$tmp_dir/conductor.log"
conductor_args_file="$tmp_dir/conductor.args"
SMOKE_ARGS_FILE="$conductor_args_file" SMOKE_FAKE_SLEEP=30 \
CONDUCTOR_BIN="$bin_dir/fake-conductor" \
  "$start_conductor" \
  --background \
  --config "$config_file" \
  --root "$runs_root" \
  --host 127.0.0.1 \
  --port 19082 \
  --pid-file "$conductor_pid_file" \
  --log-file "$conductor_log_file" >/dev/null
[ -f "$conductor_pid_file" ] || fail "conductor pid file not created"
conductor_pid="$(cat "$conductor_pid_file")"
kill -0 "$conductor_pid" >/dev/null 2>&1 || fail "conductor fake process is not running"
pids_to_kill+=("$conductor_pid")
wait_for_file "$conductor_args_file" 50 || fail "conductor args file not created"
rg -Fx -- '--config' "$conductor_args_file" >/dev/null || fail "conductor did not receive --config"
rg -Fx -- "$config_file" "$conductor_args_file" >/dev/null || fail "conductor did not receive config path"

log "3/4 validating dry-run output for monitor startup wrapper"
monitor_dry="$tmp_dir/monitor-dry.txt"
"$start_monitor" \
  --dry-run \
  --bin "$bin_dir/fake-run-agent" \
  --config "$config_file" \
  --root "$runs_root" \
  --host 127.0.0.1 \
  --port 19083 >"$monitor_dry"
rg -q 'run-agent serve startup configuration' "$monitor_dry" || fail "missing monitor header in dry-run output"
rg -q 'Task execution: disabled' "$monitor_dry" || fail "monitor dry-run should disable task execution by default"

log "4/4 validating background mode for monitor startup wrapper"
monitor_pid_file="$tmp_dir/monitor.pid"
monitor_log_file="$tmp_dir/monitor.log"
monitor_args_file="$tmp_dir/monitor.args"
SMOKE_ARGS_FILE="$monitor_args_file" SMOKE_FAKE_SLEEP=30 \
  "$start_monitor" \
  --background \
  --bin "$bin_dir/fake-run-agent" \
  --config "$config_file" \
  --root "$runs_root" \
  --host 127.0.0.1 \
  --port 19084 \
  --pid-file "$monitor_pid_file" \
  --log-file "$monitor_log_file" >/dev/null
[ -f "$monitor_pid_file" ] || fail "monitor pid file not created"
monitor_pid="$(cat "$monitor_pid_file")"
kill -0 "$monitor_pid" >/dev/null 2>&1 || fail "monitor fake process is not running"
pids_to_kill+=("$monitor_pid")
wait_for_file "$monitor_args_file" 50 || fail "monitor args file not created"
rg -Fx -- 'serve' "$monitor_args_file" >/dev/null || fail "run-agent did not receive serve subcommand"
rg -Fx -- '--disable-task-start' "$monitor_args_file" >/dev/null || fail "run-agent did not receive --disable-task-start"

log "PASS startup script smoke checks completed"
