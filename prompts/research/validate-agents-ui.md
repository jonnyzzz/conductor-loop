# Validation Task: Agent Backends & UI Facts

You are a validation agent. Cross-check existing facts against actual source code.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: agent-backend, ui, <agent-name>]
<fact text>

```

## Step 1: Read existing facts
`cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md`

## Step 2: Verify agent CLI flags against actual source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Verify exact Claude CLI flags
grep -rn "claude\|Claude" internal/agent/ --include="*.go" -l
cat internal/agent/claude/claude.go 2>/dev/null | head -60
grep -n "\-p\|input-format\|output-format\|tools\|permission" internal/agent/ -r --include="*.go" | head -20

# Verify Codex CLI flags
cat internal/agent/codex/codex.go 2>/dev/null | head -60
grep -n "dangerously\|-C\|codex" internal/agent/ -r --include="*.go" | head -20

# Verify Gemini CLI flags
cat internal/agent/gemini/gemini.go 2>/dev/null | head -60
grep -n "screen-reader\|approval-mode\|yolo" internal/agent/ -r --include="*.go" | head -20

# Verify env var names for each agent
grep -n "ANTHROPIC_API_KEY\|OPENAI_API_KEY\|GEMINI_API_KEY\|PERPLEXITY_API_KEY\|XAI_API_KEY" internal/ -r --include="*.go" | head -20

# Verify version detection
grep -n "version\|Version\|--version\|parseVersion" internal/runner/validate.go | head -30

# Verify min version constraints
grep -n "minVersion\|claude.*1\.\|codex.*0\.\|gemini.*0\." internal/runner/ -r --include="*.go" | head -10

# Verify agent selection / round-robin
grep -n "round.robin\|RoundRobin\|weighted\|fallback\|diversif" internal/ -r --include="*.go" | head -20

# Verify UI technology
ls frontend/src/ 2>/dev/null | head -10
cat frontend/package.json 2>/dev/null | head -20
ls web/src/ 2>/dev/null | head -10

# Verify web UI default port
grep -n "14355\|8080\|port" internal/api/ -r --include="*.go" | head -10
grep -n "14355\|8080" cmd/ -r --include="*.go" | head -10

# Check SSE endpoint paths
grep -rn "stream\|/sse\|EventSource\|text/event-stream" internal/api/ --include="*.go" | head -20

# Read UI spec history
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-monitoring-ui.md | head -10

# Check actual API routes
grep -rn "HandleFunc\|router\.\|mux\." internal/api/ --include="*.go" | head -30
```

## Step 3: Read ALL revisions of agent backend specs
```bash
cd /Users/jonnyzzz/Work/conductor-loop
for agent in claude codex gemini perplexity xai; do
  echo "=== $agent ==="
  git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/specifications/subsystem-agent-backend-${agent}.md | head -5
done
```

## Step 4: Write corrected output
Add section: `## Validation Round 2 (gemini)` for new/corrected entries.
