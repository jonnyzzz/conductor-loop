# Problem: Message Bus Race Condition → Data Loss

## Context
The current specification says: "run-agent bus writes via temp file + atomic swap" (subsystem-message-bus-tools.md:113).

## Problem
This "read-modify-write" strategy is catastrophically broken for concurrent appends:

1. Agent A reads MESSAGE-BUS.md (10KB)
2. Agent B reads MESSAGE-BUS.md (10KB)
3. Agent A writes MESSAGE-BUS.md.tmp (10KB + Message A), renames to MESSAGE-BUS.md
4. Agent B writes MESSAGE-BUS.md.tmp (10KB + Message B), renames to MESSAGE-BUS.md
5. **Result: Message A is permanently lost**

## Impact
Loss of critical orchestration messages (DONE, STOP, FACT updates) will break Ralph loop logic and task coordination.

## Reviewer Consensus
**ALL 6 agents** (3 Claude + 3 Gemini) independently identified this as the #1 critical blocker.

## Your Task
Propose a concrete solution for atomic message appends that prevents data loss. Consider:

1. **O_APPEND + flock**: POSIX atomic append with file locking
2. **Lock service**: Dedicated coordination process
3. **Sequence numbers**: Detect and recover from conflicts
4. **Alternative architecture**: Different message bus implementation

For your chosen solution:
- Specify the exact Go implementation approach
- Define the write algorithm (pseudocode or Go)
- Address edge cases (file permissions, crash during write, concurrent reads)
- Consider performance for long message bus files (100s of messages)
- Maintain compatibility with CLI + REST API access

## Constraints
- Must work with file-based storage (no database)
- Must support both CLI invocation and REST API
- Must handle crashes/restarts gracefully
- Should not require external services (Redis, etc.)

Provide a clear recommendation with implementation details.
