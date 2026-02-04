#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
RUNS_DIR="${RUNS_DIR:-$BASE_DIR/runs}"
LOG_FILE="$RUNS_DIR/agent-watch.log"
INTERVAL="${INTERVAL:-60}"

mkdir -p "$RUNS_DIR"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] watch-agents starting (runs=$RUNS_DIR interval=${INTERVAL}s)" >> "$LOG_FILE"

while true; do
  ts="$(date '+%Y-%m-%d %H:%M:%S')"
  running=0
  finished=0
  unknown=0

  for run_dir in "$RUNS_DIR"/run_*; do
    [ -d "$run_dir" ] || continue
    run_id="$(basename "$run_dir")"
    pid_file="$run_dir/pid.txt"
    cwd_file="$run_dir/cwd.txt"

    status="unknown"
    if [ -f "$pid_file" ]; then
      pid="$(cat "$pid_file" 2>/dev/null || true)"
      if [ -n "$pid" ] && ps -p "$pid" >/dev/null 2>&1; then
        status="running"
      fi
    elif [ -f "$cwd_file" ] && grep -q '^EXIT_CODE=' "$cwd_file"; then
      status="finished"
    fi

    case "$status" in
      running) running=$((running + 1)) ;;
      finished) finished=$((finished + 1)) ;;
      *) unknown=$((unknown + 1)) ;;
    esac

    echo "[$ts] $run_id $status" >> "$LOG_FILE"
  done

  echo "[$ts] SUMMARY running=$running finished=$finished unknown=$unknown" >> "$LOG_FILE"
  sleep "$INTERVAL"
done
