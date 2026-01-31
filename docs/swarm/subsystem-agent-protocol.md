# Agent Protocol & Governance Subsystem

## Overview
Defines behavioral rules for all agents in the swarm, including delegation, communication, state/fact updates, and git safety.

## Goals
- Ensure small, focused agent runs that exit on completion.
- Force all communication through message bus files.
- Maintain a reliable state/facts trail for handoff.
- Prevent accidental git changes outside the task scope.

## Non-Goals
- Enforcing model-specific behaviors.
- Replacing human oversight for high-risk changes.

## Behavioral Rules
- Agents MUST work on a scoped task and exit when done.
- Agents MUST delegate if a task is too large.
- Agents MUST read TASK_STATE.md and MESSAGE-BUS on start.
- Agents MUST write updates to TASK_STATE.md each cycle.
- Agents MUST log durable facts into FACT-*.md files.
- Agents MUST NOT communicate directly with other agents; use message bus.
- Agents SHOULD rotate agent type across restarts to reduce bias/stalls.

## Required Artifacts
- TASK_STATE.md: current status and next steps.
- FACT-*.md: durable knowledge.
- MESSAGE-BUS.md: questions, decisions, user replies.

## Git Safety Requirements
- Agents SHOULD stage only the files they modify.
- Agents MUST NOT revert unrelated changes.
- Agents SHOULD avoid destructive commands (reset --hard, checkout --).
- Agents SHOULD provide a clear list of touched files.

## Delegation Rules
- If task cannot be completed within a single run, delegate sub-tasks.
- Sub agents inherit the same communication and state rules.
- Parent agent records delegation decisions in TASK_STATE.md.

## Exit Conditions
- Task completed (status: done in TASK_STATE.md).
- Task blocked (status: blocked + reason).
- Required inputs missing (post question to MESSAGE-BUS and exit).

## Error Handling
- On error, write to ISSUES.md and update TASK_STATE.md.
- If message bus is unavailable, create it and retry.
