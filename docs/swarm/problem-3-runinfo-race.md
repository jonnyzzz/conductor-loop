# Problem: run-info.yaml Update Race Condition

## Context
From subsystem-storage-layout-run-info-schema.md:18:
> "Written at run start; updated at run end"

From AGGREGATED-REVIEW-FEEDBACK.md:
> "unclear if this is atomic rewrite or in-place update"

## Problem
The specification doesn't define HOW run-info.yaml is updated when a run completes:

1. **In-place update?**
   - Seek to specific fields and overwrite?
   - Risk: UI reading mid-update sees corrupt/partial YAML

2. **Atomic rewrite?**
   - Write to run-info.yaml.tmp, then rename?
   - But spec says "temp file + rename" is for message bus, not run-info

3. **Append-only?**
   - Write end state to separate file?
   - Doesn't match "update" language

## Impact

### Race Scenario
```
UI thread:        Runner thread:
read run-info     |
  pid: 12345      |
  status: running |
                  | update run-info
                  | - exit_code: 0
  status: run...  | - end_time: ...
  [corrupted YAM  |
```

**Consequences:**
- UI crashes on malformed YAML
- Monitoring tools see inconsistent state
- Logs may reference non-existent runs
- Race window: ~10-50ms per update

## Reviewer Consensus
**4 of 6 agents** flagged this (Claude #2, #3 + Gemini #2)

## Your Task

Propose a concrete solution for run-info.yaml updates that prevents race conditions.

Consider these approaches:

### Approach A: Atomic Rewrite (Temp + Rename)
```go
// At end of run
runInfo.ExitCode = exitCode
runInfo.EndTime = time.Now()
yamlBytes := yaml.Marshal(runInfo)
ioutil.WriteFile("run-info.yaml.tmp", yamlBytes, 0644)
os.Rename("run-info.yaml.tmp", "run-info.yaml")  // atomic
```

**Pros:**
- Zero race window (atomic rename)
- UI always sees consistent state
- Same pattern as message bus

**Cons:**
- Full rewrite even for small updates
- O(N) even though only end_time changed

### Approach B: File Locking
```go
f, _ := os.OpenFile("run-info.yaml", os.O_RDWR, 0644)
flock(f, LOCK_EX)
runInfo := readRunInfo(f)
runInfo.ExitCode = exitCode
f.Truncate(0)
f.Seek(0, 0)
writeYAML(f, runInfo)
flock(f, LOCK_UN)
```

**Pros:**
- Explicit synchronization
- Readers use LOCK_SH to wait

**Cons:**
- Readers must implement locking
- More complex than atomic rename
- Lock contention on busy systems

### Approach C: Separate End File
```go
// At start: write run-info.yaml
// At end: write run-info-END.yaml
// Readers: merge both files if END exists
```

**Pros:**
- No updates to original file
- Append-only (no races)

**Cons:**
- Readers must check two files
- Breaks run-info.yaml as single source of truth
- More complex for consumers

### Approach D: Tolerate Partial Reads
```go
// No special handling - readers must be robust
// UI/REST API catches YAML parse errors and retries
```

**Pros:**
- Simplest (no changes needed)
- Race window is tiny (~10ms)

**Cons:**
- Pushes complexity to all readers
- Monitoring tools may miss data
- Error logs polluted with transient parse failures

For your chosen approach, specify:

1. **Exact algorithm**: Pseudocode for update operation
2. **Reader changes**: What must readers do differently?
3. **Performance impact**: Cost of atomic rewrite vs in-place
4. **Error handling**: What if rename fails? Disk full?
5. **Compatibility**: Does this work with REST API streaming run-info?
6. **Specification updates**: Changes to subsystem-storage-layout-run-info-schema.md:18

## Constraints
- Must be race-free or tolerate races explicitly
- Must work with concurrent readers (UI, REST API, monitoring)
- Should not significantly impact performance
- Should be simple to implement correctly

Provide a clear recommendation with implementation details.
