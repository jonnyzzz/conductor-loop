# Architecture Decisions

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/decisions.md`.

## Content Requirements
Document the "Why" behind key architectural choices. Use a clear format (Context, Decision, Consequences/Trade-offs).

1. **Filesystem over Database**: 
    - Why: Simplicity, offline-first, git-ops friendly, zero-config. 
    - Trade-off: No complex queries, scaling limits (single node).
2. **O_APPEND + flock for Message Bus**: 
    - Why: Atomic on Unix, durable, no external dependencies (like Redis/Kafka).
    - Trade-off: Windows locking issues, file size growth.
3. **CLI-wrapped Agents (Claude/Codex/Gemini)**: 
    - Why: Process isolation, independent updates, simple I/O redirection.
    - *Note*: Perplexity/xAI use REST adapters in-process.
4. **YAML over HCL for Configuration**: 
    - Why: Better Go ecosystem support (`gopkg.in/yaml.v3`), standardization. HCL is supported for legacy only.
5. **Default Port 14355**: 
    - Why: Avoid conflicts with common ports like 8080 or 3000.
6. **Process Groups (PGID)**: 
    - Why: Clean cleanup of child processes tree.
    - Trade-off: Limited Windows support.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/` (check these files for context)

## Instructions
- Read the facts and existing decision records.
- Synthesize them into a cohesive narrative.
- Ensure the document is named `decisions.md`.
