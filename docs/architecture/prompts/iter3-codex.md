# Deployment Architecture

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/deployment.md`.

## Content Requirements
1. **Deployment Model**:
    - Single-binary (`run-agent` or `conductor`).
    - Local execution (laptop/workstation).
2. **Directory Structure**:
    - Root -> Projects -> Tasks -> Runs.
    - Config location (`~/.config/conductor/`).
3. **Operations**:
    - Garbage Collection (`run-agent gc`).
    - Self-Update mechanism (`run-agent server update`).
    - Logging (`internal/obslog`).
4. **Docker**:
    - `Dockerfile` and `docker-compose.yml` support.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

## Instructions
- Explain how the software is deployed and maintained.
- Name the file `deployment.md`.
