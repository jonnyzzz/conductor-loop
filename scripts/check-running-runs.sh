#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage: check-running-runs.sh <root_path> <project_id>

Scans the latest run per task under <root_path>/<project_id>/task-* and prints
only runs that are currently marked as status: running.
EOF
}

if [[ $# -ne 2 ]]; then
    usage
    exit 1
fi

root_path=$1
project_id=$2
project_dir="$root_path/$project_id"

if [[ ! -d "$project_dir" ]]; then
    echo "Project directory does not exist: $project_dir" >&2
    exit 1
fi

printf 'Running task runs (latest per task) for %s\n' "$project_dir"
printf '%-42s %-30s %-8s %-5s %-9s %-4s\n' "task_id" "run_id" "pid" "alive" "exit_code" "done"

found=0

for task_dir in "$project_dir"/task-*; do
    [[ -d "$task_dir" ]] || continue

    runs_dir="$task_dir/runs"
    [[ -d "$runs_dir" ]] || continue

    run_id=$(ls -1 "$runs_dir" 2>/dev/null | tail -n 1)
    [[ -n "$run_id" ]] || continue

    info_path="$runs_dir/$run_id/run-info.yaml"
    [[ -f "$info_path" ]] || continue

    status=$(awk -F': ' '/^status:/{print $2; exit}' "$info_path")
    [[ "$status" == "running" ]] || continue

    pid=$(awk -F': ' '/^pid:/{print $2; exit}' "$info_path")
    exit_code=$(awk -F': ' '/^exit_code:/{print $2; exit}' "$info_path")

    alive="no"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
        alive="yes"
    fi

    done_flag="no"
    if [[ -f "$task_dir/DONE" ]]; then
        done_flag="yes"
    fi

    printf '%-42s %-30s %-8s %-5s %-9s %-4s\n' \
        "$(basename "$task_dir")" "$run_id" "$pid" "$alive" "$exit_code" "$done_flag"
    found=1
done

if [[ $found -eq 0 ]]; then
    echo "No running task runs found"
fi
