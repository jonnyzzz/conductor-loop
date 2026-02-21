# Problem #1 Decision: Message Bus Race Condition

## Three Agent Proposals

### Agent 1 (Claude)
**Recommendation**: O_APPEND + flock solution
- Zero data loss under concurrent writes
- Crash-safe (locks auto-released)
- Simple implementation (~150 lines Go)
- No external dependencies
- Acceptable performance (500+ msg/s)
- Works with CLI and REST API

### Agent 2 (Codex)
**Recommendation**: O_APPEND + flock with record framing
- Binary framing: `[8 bytes length][4 bytes CRC32][payload]`
- OR human-readable framing with delimiters
- fsync for durability
- Crash-safe with CRC validation
- Handles partial records on crashes

### Agent 3 (Gemini)
**Recommendation**: O_APPEND + flock (append-only)
- Standard POSIX solution
- `LOCK_EX` for writers, `LOCK_SH` for readers
- fsync for crash safety
- OS automatically releases locks on crash
- Fast performance (syscall-level locking)

## Consensus
**ALL 3 AGENTS** recommend O_APPEND + flock as the solution.

## Key Decision Points

1. **Record Framing**: Should we use?
   - Agent 2 suggests binary/text framing for crash recovery
   - Agents 1 & 3 suggest simple append without framing

2. **Shared Locks for Readers**: Should we use?
   - Agent 3 suggests `LOCK_SH` for readers to prevent reading mid-write
   - Agents 1 & 2 don't mention it

3. **fsync Requirement**: Should we always fsync?
   - All agents suggest fsync for durability
   - Performance trade-off: ~100x slower but crash-safe

## Your Task

Make the final decision and specify:

1. **Exact algorithm**: Detailed pseudocode for append operation
2. **Record format**: Keep current YAML format or add framing?
3. **Read locking**: Use LOCK_SH for readers or allow unlocked reads?
4. **fsync policy**: Always fsync or make it optional/configurable?
5. **Error handling**: What to do on lock timeout, write failure, etc.?
6. **Specification updates**: What exact changes to subsystem-message-bus-tools.md?

Provide a clear, implementable decision that resolves all ambiguities.
