# Docs Update R3: Dev Docs Deep Re-check

You are a documentation update agent (Round 3). Dev docs were updated in Round 2.
Your job is to catch anything missed — especially from FACTS-runs-conductor.md (125 run history).

## Files to deep-check and fix

1. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/ralph-loop.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/message-bus.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/storage-layout.md`
6. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/agent-protocol.md`
7. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/adding-agents.md`

## Facts sources (read ALL — focus on runs facts)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-messagebus.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md
```

## Verify newly shipped features against docs

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Task dependencies: depends_on — in docs?
grep -n "depends_on\|DependsOn\|dependency\|depends" docs/dev/ -r | head -10
grep -n "depends_on\|DependsOn" internal/ -r --include="*.go" | head -10

# Webhook notifications — in docs?
grep -n "webhook\|Webhook" docs/dev/ -r | head -10
grep -n "webhook\|Webhook" internal/ -r --include="*.go" | head -5

# Task completion fact propagation — in docs?
grep -n "fact.*propag\|propag.*fact\|FACT.*task\|PROJECT-MESSAGE-BUS" docs/dev/ -r | head -10

# Run-state liveness healing — in docs?
grep -n "liveness\|reconcile\|Reconcile\|stale.*run\|dead.*pid" docs/dev/ -r | head -10

# Form submission audit log — in docs?
grep -n "audit\|form.*submit\|JSONL\|_audit" docs/dev/ -r | head -10

# Prometheus metrics — in docs?
grep -n "prometheus\|metrics\|/metrics" docs/dev/ -r | head -10

# Root-task planner queue — in docs?
grep -n "planner\|root.*task.*queue\|max_concurrent_root" docs/dev/ -r | head -10
```

## Rules

- **Facts override docs**
- For each feature found in README.md features list but missing from dev docs: add a brief section
- Key features from README that dev docs may lack:
  - Task dependencies (`depends_on`)
  - Webhook notifications config
  - Task completion fact propagation to project bus
  - Run-state liveness healing (dead PID reconciliation)
  - Form submission audit log
  - Prometheus `/metrics` endpoint
  - Root-task planner queue (`max_concurrent_root_tasks`)
  - Agent diversification policy (`DiversificationConfig`)
- Only add what is verified in code — no speculation
- Keep additions concise — one paragraph or small table per feature

## Output

Overwrite each file in-place. Write summary to `output.md`.
