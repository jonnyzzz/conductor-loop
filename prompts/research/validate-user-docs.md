# Validation Task: User & Developer Documentation Facts

You are a validation agent. Cross-check existing facts against actual CLI help output and source code.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md`

Format:
```
[YYYY-MM-DD HH:MM:SS] [tags: user-docs, dev-docs, cli, config, <topic>]
<fact text>

```

## Step 1: Read existing facts
`cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md`

## Step 2: Verify CLI commands against actual binary

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Get ACTUAL help output from the binary
./bin/run-agent --help
./bin/run-agent job --help
./bin/run-agent task --help
./bin/run-agent bus --help
./bin/run-agent bus post --help
./bin/run-agent bus read --help
./bin/run-agent status --help
./bin/run-agent list --help
./bin/run-agent watch --help
./bin/run-agent gc --help
./bin/run-agent validate --help
./bin/run-agent serve --help
./bin/run-agent workflow --help
./bin/run-agent workflow run --help
./bin/run-agent goal --help
./bin/run-agent goal decompose --help
./bin/run-agent job batch --help
./bin/run-agent output --help
./bin/run-agent resume --help
./bin/run-agent stop --help
./bin/run-agent wrap --help
./bin/run-agent shell-setup --help
./bin/run-agent monitor --help
./bin/run-agent iterate 2>&1 | head -10

# Also check conductor binary
./bin/conductor --help 2>&1 | head -30

# Verify default port
grep -rn "14355\|DefaultPort\|default.*port" cmd/ internal/api/ --include="*.go" | head -10

# Verify config file search paths
grep -rn "config.yaml\|config.hcl\|\.conductor\|XDG_CONFIG" cmd/ internal/config/ --include="*.go" | head -20

# Read actual config schema
cat internal/config/config.go

# Verify installation script
cat install.sh | head -40

# Verify run-agent.cmd launcher
cat run-agent.cmd | head -30

# Check git log for user docs
git log --format="%H %ad %s" --date=format:"%Y-%m-%d %H:%M:%S" -- docs/user/ README.md | head -20

# Read README.md history - each revision
for sha in $(git log --format="%H" -- README.md | head -5); do
  date=$(git log -1 --format="%ad" --date=format:"%Y-%m-%d %H:%M:%S" $sha)
  echo "=== README $date ==="
  git show $sha:README.md | head -30
done

# Read current user docs
cat docs/user/cli-reference.md | head -80
cat docs/user/configuration.md | head -60
cat docs/user/installation.md | head -40
cat docs/user/quick-start.md | head -40
```

## Step 3: Verify API endpoints against actual code
```bash
cd /Users/jonnyzzz/Work/conductor-loop
# Get all routes
grep -rn "HandleFunc\|\.Handle\|GET\|POST\|DELETE\|PUT" internal/api/ --include="*.go" | grep -v test | head -40
cat internal/api/routes.go 2>/dev/null || find internal/api -name "*.go" | head -5
```

## Step 4: Write corrected output
Focus on precision: exact flag names, exact default values, exact port numbers, exact API paths.
Add section: `## Validation Round 2 (codex)` for new/corrected entries.
