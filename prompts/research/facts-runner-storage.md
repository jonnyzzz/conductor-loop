# Research Task: Runner, Storage & Environment Facts

You are a research agent. Extract key facts from runner and storage specification documents, tracing their evolution through git history.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: runner, storage, <subsystem>]
<fact text — concrete decision, spec value, or implementation detail>

```

## Files to Research

### Specifications:
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-config-schema.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout-run-info-schema.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-env-contract.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-env-contract-QUESTIONS.md`

### Dev docs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/storage-layout.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/agent-protocol.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/self-update.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/release-checklist.md`

### Legacy swarm specs (earlier versions):
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-runner-orchestration.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-runner-orchestration-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-runner-orchestration-config-schema.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-storage-layout.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-storage-layout-QUESTIONS.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-storage-layout-run-info-schema.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-env-contract.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/swarm/docs/legacy/subsystem-env-contract-QUESTIONS.md`

## Instructions

1. For each file, get git history: `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- <file>`

2. Read each revision that changed significantly: `git show <sha>:<file>`

3. Compare legacy swarm specs vs current conductor-loop specs to track evolution

4. Extract facts: storage paths, run ID formats, timeout values, config schema fields, environment variable names, Ralph loop parameters, idle/stuck thresholds, DONE marker semantics, restart policies, GC parameters

5. For questions files, note which questions were answered and what the answers were

6. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`

## Start now.
