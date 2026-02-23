# Validation Task: Message Bus & Agent Protocol Facts

You are a validation agent. Cross-check existing facts against source code and git history.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: messagebus, agent-protocol, <subsystem>]
<fact text>

```

## Step 1: Read existing facts
`cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md`

## Step 2: Verify against actual source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Verify message format (YAML front-matter)
cat internal/messagebus/messagebus.go | head -100

# Verify msg_id format
grep -n "MSG-\|msg_id\|msgID\|MsgID" internal/messagebus/ -r --include="*.go" | head -20

# Verify message types
grep -n "FACT\|QUESTION\|ANSWER\|RUN_START\|RUN_STOP\|CRASH\|ERROR\|WARNING\|INFO\|PROGRESS\|DECISION" internal/messagebus/ -r --include="*.go" | head -30

# Verify locking strategy
cat internal/messagebus/lock.go | head -40
grep -n "flock\|O_APPEND\|LockExclusive\|LockShared" internal/messagebus/ -r --include="*.go" | head -20

# Verify bus auto-discovery
grep -rn "auto.discover\|FindBus\|findBus\|MESSAGE-BUS\|PROJECT-MESSAGE-BUS\|TASK-MESSAGE-BUS" internal/ --include="*.go" | head -20

# Check bus rotation
grep -n "rotate\|Rotate\|WithAutoRotate\|maxBytes\|64KB\|64 *1024" internal/messagebus/ -r --include="*.go" | head -20

# Verify ReadLastN / tail
grep -n "ReadLastN\|tail\|--tail" internal/messagebus/ -r --include="*.go" | head -10
grep -n "ReadLastN\|tail" cmd/run-agent/ -r --include="*.go" | head -10

# Check parents[] / threading model
grep -n "parents\|Parents\|parent_id\|ParentID" internal/messagebus/ -r --include="*.go" | head -20

# Check git history for messagebus spec files
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-message-bus-tools.md docs/specifications/subsystem-message-bus-object-model.md docs/specifications/subsystem-agent-protocol.md | head -20

# Read problem-1 decision (message bus race condition)
cat docs/swarm/docs/legacy/problem-1-decision.md

# Read current message-bus dev doc
cat docs/dev/message-bus.md

# Check bus CLI commands
grep -n "bus\|Bus" cmd/run-agent/ -r --include="*.go" -l
cat cmd/run-agent/bus.go 2>/dev/null | head -60
```

## Step 3: Check jonnyzzz-ai-coder message-bus-mcp
```bash
ls /Users/jonnyzzz/Work/jonnyzzz-ai-coder/message-bus-mcp/ 2>/dev/null | head -10
cat /Users/jonnyzzz/Work/jonnyzzz-ai-coder/message-bus-mcp/SPEC.md 2>/dev/null | head -50
```

## Step 4: Read ALL revisions of message bus spec
```bash
cd /Users/jonnyzzz/Work/conductor-loop
git log --format="%H %ad" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-message-bus-tools.md
# Read first revision
FIRST=$(git log --format="%H" -- docs/specifications/subsystem-message-bus-tools.md | tail -1)
FIRST_DATE=$(git log --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-message-bus-tools.md | tail -1)
echo "First: $FIRST_DATE"
git show $FIRST:docs/specifications/subsystem-message-bus-tools.md | head -60
```

## Step 5: Write corrected output
Add section: `## Validation Round 2 (codex)` for new/corrected entries.
