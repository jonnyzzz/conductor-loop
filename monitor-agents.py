#!/usr/bin/env python3
import argparse
import os
import sys
import time
from pathlib import Path
from typing import Dict, Tuple


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Live monitor for agent runs")
    parser.add_argument("--runs-dir", default=None, help="Runs directory (defaults to RUNS_DIR env or ./runs)")
    parser.add_argument("--poll-interval", type=float, default=0.5, help="Polling interval in seconds")
    parser.add_argument("--summary-interval", type=float, default=5.0, help="Summary header interval in seconds")
    return parser.parse_args()


def resolve_runs_dir(args: argparse.Namespace) -> Path:
    if args.runs_dir:
        return Path(args.runs_dir).expanduser().resolve()
    env_dir = os.environ.get("RUNS_DIR")
    if env_dir:
        return Path(env_dir).expanduser().resolve()
    return (Path(__file__).resolve().parent / "runs").resolve()


def read_exit_code(cwd_file: Path) -> bool:
    try:
        for line in cwd_file.read_text(errors="replace").splitlines():
            if line.startswith("EXIT_CODE="):
                return True
    except FileNotFoundError:
        return False
    return False


def detect_status(run_dir: Path) -> str:
    pid_file = run_dir / "pid.txt"
    if pid_file.exists():
        return "running"
    if read_exit_code(run_dir / "cwd.txt"):
        return "finished"
    return "unknown"


def list_runs(runs_dir: Path) -> Tuple[Path, ...]:
    if not runs_dir.exists():
        return tuple()
    runs = [p for p in runs_dir.iterdir() if p.is_dir() and p.name.startswith("run_")]
    return tuple(sorted(runs, key=lambda p: p.name))


def print_summary(runs_dir: Path) -> None:
    running = finished = unknown = 0
    for run_dir in list_runs(runs_dir):
        status = detect_status(run_dir)
        if status == "running":
            running += 1
        elif status == "finished":
            finished += 1
        else:
            unknown += 1
    ts = time.strftime("%Y-%m-%d %H:%M:%S")
    sys.stdout.write(f"[{ts}] runs: running={running} finished={finished} unknown={unknown}\n")
    sys.stdout.flush()


def read_new_text(path: Path, offset: int) -> Tuple[str, int]:
    try:
        size = path.stat().st_size
    except FileNotFoundError:
        return "", offset
    if size < offset:
        offset = 0
    with path.open("r", errors="replace") as handle:
        handle.seek(offset)
        data = handle.read()
        return data, handle.tell()


def main() -> int:
    args = parse_args()
    runs_dir = resolve_runs_dir(args)

    offsets: Dict[Path, int] = {}
    buffers: Dict[Path, str] = {}

    print_summary(runs_dir)
    last_summary = time.time()

    while True:
        now = time.time()
        if now - last_summary >= args.summary_interval:
            print_summary(runs_dir)
            last_summary = now

        for run_dir in list_runs(runs_dir):
            run_id = run_dir.name
            for stream_name, filename in ("stdout", "agent-stdout.txt"), ("stderr", "agent-stderr.txt"):
                path = run_dir / filename
                offset = offsets.get(path, 0)
                data, offset = read_new_text(path, offset)
                offsets[path] = offset
                if not data:
                    continue
                buffered = buffers.get(path, "") + data
                lines = buffered.splitlines(keepends=True)
                if lines and not lines[-1].endswith("\n"):
                    buffers[path] = lines[-1]
                    lines = lines[:-1]
                else:
                    buffers[path] = ""
                for line in lines:
                    sys.stdout.write(f"[{run_id}] {stream_name}: {line}")
                sys.stdout.flush()

        time.sleep(args.poll_interval)


if __name__ == "__main__":
    raise SystemExit(main())
