---
title: "Message Bus Protocol"
description: "Operational contract for FACT/PROGRESS/DECISION/ERROR messages"
weight: 40
group: "Core Docs"
---

The message bus is the project and task coordination stream used by agents and tooling.

## Message Types

- `FACT`: concrete outcomes (test result, commit hash, output path)
- `PROGRESS`: in-flight status updates
- `DECISION`: explicit choice with rationale
- `ERROR`: blocking issue with remediation context
- `QUESTION`: open item needing response
- `INFO`: additional non-blocking context

## Posting Messages

Use the CLI helper:

```bash
./run-agent bus post --type PROGRESS --body "Reading storage subsystem"
./run-agent bus post --type FACT --body "go test ./... passed"
```

When project-aware context is available:

```bash
./run-agent bus post \
  --project "$JRUN_PROJECT_ID" \
  --task "$JRUN_TASK_ID" \
  --root "$CONDUCTOR_ROOT" \
  --type DECISION \
  --body "Split work into independent backend and API tracks"
```

## More Details

- Developer doc: [`docs/dev/message-bus.md`](https://github.com/jonnyzzz/conductor-loop/blob/main/docs/dev/message-bus.md)
- API details: [`docs/user/api-reference.md`](https://github.com/jonnyzzz/conductor-loop/blob/main/docs/user/api-reference.md)
