# Problem: SSE Stream Run Discovery Missing

## Context
From subsystem-frontend-backend-api.md:285:
> "New runs automatically included" in log stream

## Problem
No specification for HOW backend discovers new run folders.

## Options
1. **inotify** (Linux) / FSEvents (macOS) - Filesystem watching
2. **Polling** - Check runs/ directory every N seconds
3. **Message Bus** - Listen for RUN_STARTED messages

## Your Task
Choose discovery mechanism and specify:
- Exact algorithm for detecting new runs
- How to handle run directory creation mid-stream
- Performance implications
- Cross-platform compatibility (Linux/macOS/Windows)

Recommend approach with implementation details.
