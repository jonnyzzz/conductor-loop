# Research Task: User & Developer Documentation Facts

You are a research agent. Extract key facts from user-facing and developer documentation, tracing their evolution through git history.

## Output Format

Write all facts to: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md`

Each fact entry MUST follow this exact format:
```
[YYYY-MM-DD HH:MM:SS] [tags: user-docs, dev-docs, cli, config, <topic>]
<fact text — CLI command syntax, config field, installation step, or API endpoint>

```

## Files to Research

### User docs (ALL revisions):
- `/Users/jonnyzzz/Work/conductor-loop/README.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/development.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/installation.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/quick-start.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/configuration.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/api-reference.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/web-ui.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/troubleshooting.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/faq.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/rlm-orchestration.md`

### Dev docs:
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/development-setup.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/message-bus.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/logging-observability.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/dev/documentation-site.md`

### Config examples:
- `/Users/jonnyzzz/Work/conductor-loop/config.local.yaml`
- `/Users/jonnyzzz/Work/conductor-loop/config.docker.yaml`
- Run: `cat /Users/jonnyzzz/Work/conductor-loop/config.local.yaml`
- Run: `cat /Users/jonnyzzz/Work/conductor-loop/config.docker.yaml`

## Instructions

1. Get git history for each file:
   `cd /Users/jonnyzzz/Work/conductor-loop && git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- README.md docs/dev/development.md docs/user/ docs/dev/`

2. Read current state of each file and key historical revisions

3. Extract facts:
   - Installation commands (brew, curl, go install)
   - Default port (14355), config file locations
   - All CLI commands and their flags (run-agent, conductor)
   - Config YAML schema fields and defaults
   - All API endpoints with methods and paths
   - Web UI URL and features
   - Build commands (go build, make)
   - Test commands (go test, make test)
   - Known issues / troubleshooting tips
   - Agent minimum version requirements

4. Write ALL facts to `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md`

## Start now.
